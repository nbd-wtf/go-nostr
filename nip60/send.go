package nip60

import (
	"context"
	"encoding/hex"
	"fmt"
	"slices"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut02"
	"github.com/elnosh/gonuts/cashu/nuts/nut10"
	"github.com/elnosh/gonuts/cashu/nuts/nut11"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip60/client"
)

type SendOption func(opts *sendSettings)

type sendSettings struct {
	specificMint   string
	p2pk           *btcec.PublicKey
	refundtimelock int64
}

func WithP2PK(pubkey string) SendOption {
	return func(opts *sendSettings) {
		pkb, _ := hex.DecodeString(pubkey)
		opts.p2pk, _ = btcec.ParsePubKey(pkb)
	}
}

func WithRefundable(timelock nostr.Timestamp) SendOption {
	return func(opts *sendSettings) {
		opts.refundtimelock = int64(timelock)
	}
}

func WithMint(url string) SendOption {
	return func(opts *sendSettings) {
		opts.specificMint = url
	}
}

type chosenTokens struct {
	mint         string
	tokens       []Token
	tokenIndexes []int
	proofs       cashu.Proofs
	keysets      []nut02.Keyset
}

func (w *Wallet) SendToken(ctx context.Context, amount uint64, opts ...SendOption) (string, error) {
	ss := &sendSettings{}
	for _, opt := range opts {
		opt(ss)
	}

	w.tokensMu.Lock()
	defer w.tokensMu.Unlock()

	chosen, _, err := w.getProofsForSending(ctx, amount, ss.specificMint)
	if err != nil {
		return "", err
	}

	swapOpts := make([]SwapOption, 0, 2)

	if ss.p2pk != nil {
		if info, err := client.GetMintInfo(ctx, chosen.mint); err != nil || !info.Nuts.Nut11.Supported {
			return "", fmt.Errorf("mint doesn't support p2pk: %w", err)
		}

		tags := nut11.P2PKTags{
			NSigs:    1,
			Locktime: 0,
			Pubkeys:  []*btcec.PublicKey{ss.p2pk},
		}
		if ss.refundtimelock != 0 {
			tags.Refund = []*btcec.PublicKey{w.PublicKey}
			tags.Locktime = ss.refundtimelock
		}

		swapOpts = append(swapOpts, WithSpendingCondition(
			nut10.SpendingCondition{
				Kind: nut10.P2PK,
				Data: hex.EncodeToString(ss.p2pk.SerializeCompressed()),
				Tags: nut11.SerializeP2PKTags(tags),
			},
		))
	}

	// get new proofs
	proofsToSend, changeProofs, err := w.SwapProofs(ctx, chosen.mint, chosen.proofs, amount, swapOpts...)
	if err != nil {
		return "", err
	}

	if err := w.saveChangeAndDeleteUsedTokens(ctx, chosen.mint, changeProofs, chosen.tokenIndexes); err != nil {
		return "", err
	}

	// serialize token we're sending out
	token, err := cashu.NewTokenV4(proofsToSend, chosen.mint, cashu.Sat, true)
	if err != nil {
		return "", err
	}

	wevt := nostr.Event{}
	w.toEvent(ctx, w.wl.kr, &wevt)
	w.wl.Changes <- wevt

	return token.Serialize()
}

func (w *Wallet) saveChangeAndDeleteUsedTokens(
	ctx context.Context,
	mintURL string,
	changeProofs cashu.Proofs,
	usedTokenIndexes []int,
) error {
	// delete spent tokens and save our change
	updatedTokens := make([]Token, 0, len(w.Tokens))

	changeToken := Token{
		mintedAt: nostr.Now(),
		Mint:     mintURL,
		Proofs:   changeProofs,
		Deleted:  make([]string, 0, len(usedTokenIndexes)),
		event:    &nostr.Event{},
	}

	for i, token := range w.Tokens {
		if slices.Contains(usedTokenIndexes, i) {
			if token.event != nil {
				token.Deleted = append(token.Deleted, token.event.ID)

				deleteEvent := nostr.Event{
					CreatedAt: nostr.Now(),
					Kind:      5,
					Tags:      nostr.Tags{{"e", token.event.ID}, {"k", "7375"}},
				}
				w.wl.kr.SignEvent(ctx, &deleteEvent)
				w.wl.Changes <- deleteEvent
			}
			continue
		}
		updatedTokens = append(updatedTokens, token)
	}

	if len(changeToken.Proofs) > 0 {
		if err := changeToken.toEvent(ctx, w.wl.kr, w.Identifier, changeToken.event); err != nil {
			return fmt.Errorf("failed to make change token: %w", err)
		}
		w.wl.Changes <- *changeToken.event
		w.Tokens = append(updatedTokens, changeToken)
	}

	return nil
}

func (w *Wallet) getProofsForSending(
	ctx context.Context,
	amount uint64,
	specificMint string,
) (chosenTokens, uint64, error) {
	byMint := make(map[string]chosenTokens)
	for t, token := range w.Tokens {
		if specificMint != "" && token.Mint != specificMint {
			continue
		}

		part, ok := byMint[token.Mint]
		if !ok {
			keysets, err := client.GetAllKeysets(ctx, token.Mint)
			if err != nil {
				return chosenTokens{}, 0, fmt.Errorf("failed to get %s keysets: %w", token.Mint, err)
			}
			part.keysets = keysets
			part.tokens = make([]Token, 0, 3)
			part.tokenIndexes = make([]int, 0, 3)
			part.proofs = make(cashu.Proofs, 0, 7)
			part.mint = token.Mint
		}

		part.tokens = append(part.tokens, token)
		part.tokenIndexes = append(part.tokenIndexes, t)
		part.proofs = append(part.proofs, token.Proofs...)
		if part.proofs.Amount() >= amount {
			// maybe we found it here
			fee := calculateFee(part.proofs, part.keysets)
			if part.proofs.Amount() >= (amount + fee) {
				// yes, we did
				return part, fee, nil
			}
		}

		byMint[token.Mint] = part
	}

	// if we got here it's because we didn't get enough proofs from the same mint
	return chosenTokens{}, 0, fmt.Errorf("not enough proofs found from the same mint")
}
