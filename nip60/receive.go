package nip60

import (
	"context"
	"fmt"
	"slices"

	"github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut10"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip60/client"
)

func (w *Wallet) Receive(
	ctx context.Context,
	proofs cashu.Proofs,
	mint string,
) error {
	if w.PublishUpdate == nil {
		return fmt.Errorf("can't do write operations: missing PublishUpdate function")
	}

	source := "http" + nostr.NormalizeURL(mint)[2:]
	lightningSwap := slices.Contains(w.Mints, source)
	swapOpts := make([]SwapOption, 0, 1)

	for i, proof := range proofs {
		if proof.Secret != "" {
			nut10Secret, err := nut10.DeserializeSecret(proof.Secret)
			if err == nil {
				switch nut10Secret.Kind {
				case nut10.P2PK:
					swapOpts = append(swapOpts, WithSignedOutputs())
					proofs[i].Witness, err = signInput(w.PrivateKey, proof)
					if err != nil {
						return fmt.Errorf("failed to sign locked proof %d: %w", i, err)
					}
				case nut10.HTLC:
					return fmt.Errorf("HTLC token not supported yet")
				case nut10.AnyoneCanSpend:
					// ok
				}
			}
		}
	}

	sourceKeysets, err := client.GetAllKeysets(ctx, source)
	if err != nil {
		return fmt.Errorf("failed to get %s keysets: %w", source, err)
	}

	// get new proofs
	newProofs, _, err := w.swapProofs(ctx, source, proofs, proofs.Amount(), swapOpts...)
	if err != nil {
		return err
	}

	newMint := source // if we don't have to do a lightning swap then new mint will be the same as old mint

	// if we have to swap to our own mint we do it now by getting a bolt11 invoice from our mint
	// and telling the current mint to pay it
	if lightningSwap {
		for _, targetMint := range w.Mints {
			swappedProofs, err, status := lightningMeltMint(
				ctx,
				newProofs,
				source,
				sourceKeysets,
				targetMint,
			)
			if err != nil {
				if status == tryAnotherTargetMint {
					continue
				}
				if status == manualActionRequired {
					return fmt.Errorf("failed to swap (needs manual action): %w", err)
				}
				if status == nothingCanBeDone {
					return fmt.Errorf("failed to swap (nothing can be done, we probably lost the money): %w", err)
				}

				// if we get here that means we still have our proofs from the untrusted mint, so save those
				goto saveproofs
			} else {
				// everything went well
				newProofs = swappedProofs
				newMint = targetMint
				goto saveproofs
			}
		}

		// if we got here that means we ran out of our trusted mints to swap to, so save the untrusted proofs
		goto saveproofs
	}

saveproofs:
	newToken := Token{
		Mint:     newMint,
		Proofs:   newProofs,
		mintedAt: nostr.Now(),
		event:    &nostr.Event{},
	}
	if err := newToken.toEvent(ctx, w.kr, newToken.event); err != nil {
		return fmt.Errorf("failed to make new token: %w", err)
	}

	he := HistoryEntry{
		event: &nostr.Event{},
		TokenReferences: []TokenRef{
			{
				EventID:  newToken.event.ID,
				Created:  true,
				IsNutzap: false,
			},
		},
		createdAt: nostr.Now(),
		In:        true,
		Amount:    newToken.Proofs.Amount(),
	}

	w.Lock()
	w.PublishUpdate(*newToken.event, nil, &newToken, nil, false)
	if err := he.toEvent(ctx, w.kr, he.event); err == nil {
		w.PublishUpdate(*he.event, nil, nil, nil, true)
	}
	w.Unlock()

	w.tokensMu.Lock()
	w.Tokens = append(w.Tokens, newToken)
	w.tokensMu.Unlock()

	return nil
}
