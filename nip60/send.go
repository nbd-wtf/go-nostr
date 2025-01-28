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

func (w *Wallet) SendToken(ctx context.Context, amount uint64, opts ...SendOption) (string, error) {
	ss := &sendSettings{}
	for _, opt := range opts {
		opt(ss)
	}

	w.tokensMu.Lock()
	defer w.tokensMu.Unlock()

	type part struct {
		mint         string
		tokens       []Token
		tokenIndexes []int
		proofs       cashu.Proofs
		keysets      []nut02.Keyset
	}

	var target part
	byMint := make(map[string]part)
	for t, token := range w.Tokens {
		if ss.specificMint != "" && token.Mint != ss.specificMint {
			continue
		}

		part, ok := byMint[token.Mint]
		if !ok {
			keysets, err := client.GetAllKeysets(ctx, token.Mint)
			if err != nil {
				return "", fmt.Errorf("failed to get %s keysets: %w", token.Mint, err)
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
				target = part
				goto found
			}
		}
	}

	// if we got here it's because we didn't get enough proofs from the same mint
	return "", fmt.Errorf("not enough proofs found from the same mint")

found:
	swapOpts := make([]SwapOption, 0, 2)

	if ss.p2pk != nil {
		if info, err := client.GetMintInfo(ctx, target.mint); err != nil || !info.Nuts.Nut11.Supported {
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
	proofsToSend, changeProofs, err := w.SwapProofs(ctx, target.mint, target.proofs, amount, swapOpts...)
	if err != nil {
		return "", err
	}

	// delete spent tokens and save our change
	newTokens := make([]Token, 0, len(w.Tokens))
	for i, token := range w.Tokens {
		if slices.Contains(target.tokenIndexes, i) {
			continue
		}
		newTokens = append(newTokens, token)
	}
	w.Tokens = append(newTokens, Token{
		mintedAt: nostr.Now(),
		Mint:     target.mint,
		Proofs:   changeProofs,
	})

	// serialize token we're sending out
	token, err := cashu.NewTokenV4(proofsToSend, target.mint, cashu.Sat, true)
	if err != nil {
		return "", err
	}

	return token.Serialize()
}
