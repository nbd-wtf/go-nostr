package nip60

import (
	"context"
	"fmt"
	"slices"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut02"
	"github.com/elnosh/gonuts/cashu/nuts/nut03"
	"github.com/elnosh/gonuts/cashu/nuts/nut10"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip60/client"
)

func (w *Wallet) ReceiveToken(ctx context.Context, serializedToken string) error {
	token, err := cashu.DecodeToken(serializedToken)
	if err != nil {
		return err
	}

	source := "http" + nostr.NormalizeURL(token.Mint())[2:]
	swap := slices.Contains(w.Mints, source)
	proofs := token.Proofs()
	isp2pk := false

	for i, proof := range proofs {
		if proof.Secret != "" {
			nut10Secret, err := nut10.DeserializeSecret(proof.Secret)
			if err != nil {
				return fmt.Errorf("invalid nip10 secret at %d: %w", i, err)
			}
			switch nut10Secret.Kind {
			case nut10.P2PK:
				isp2pk = true
				proofs[i].Witness, err = signInput(w.PrivateKey, w.PublicKey, proof, nut10Secret)
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

	sourceKeysets, err := client.GetAllKeysets(ctx, source)
	if err != nil {
		return fmt.Errorf("failed to get %s keysets: %w", source, err)
	}
	var sourceActiveKeyset nut02.Keyset
	var sourceActiveKeys map[uint64]*btcec.PublicKey
	for _, keyset := range sourceKeysets {
		if keyset.Unit == cashu.Sat.String() && keyset.Active {
			sourceActiveKeyset = keyset
			sourceActiveKeysHex, err := client.GetKeysetById(ctx, source, keyset.Id)
			if err != nil {
				return fmt.Errorf("failed to get keyset keys for %s: %w", keyset.Id, err)
			}
			sourceActiveKeys, err = parseKeysetKeys(sourceActiveKeysHex)
		}
	}

	// get new proofs
	splits := make([]uint64, len(proofs))
	for i, p := range proofs { // TODO: do the fee stuff here because it won't always be free
		splits[i] = p.Amount
	}

	outputs, secrets, rs, err := createBlindedMessages(splits, sourceActiveKeyset.Id)
	if err != nil {
		return fmt.Errorf("failed to create blinded message: %w", err)
	}

	if isp2pk {
		for i, output := range outputs {
			outputs[i].Witness, err = signOutput(w.PrivateKey, output)
			if err != nil {
				return fmt.Errorf("failed to sign output message %d: %w", i, err)
			}
		}
	}

	req := nut03.PostSwapRequest{
		Inputs:  proofs,
		Outputs: outputs,
	}

	res, err := client.PostSwap(ctx, source, req)
	if err != nil {
		return fmt.Errorf("failed to swap %s->%s: %w", source, w.Mints[0], err)
	}

	newProofs, err := constructProofs(res.Signatures, req.Outputs, secrets, rs, sourceActiveKeys)
	if err != nil {
		return fmt.Errorf("failed to construct proofs: %w", err)
	}
	newMint := source

	// if we have to swap to our own mint we do it now by getting a bolt11 invoice from our mint
	// and telling the current mint to pay it
	if swap {
		for _, targetMint := range w.Mints {
			swappedProofs, err, tryAnother, needsManualAction := lightningMeltMint(
				ctx,
				newProofs,
				source,
				sourceKeysets,
				targetMint,
			)
			if err != nil {
				if tryAnother {
					continue
				}
				if needsManualAction {
					return fmt.Errorf("failed to swap (needs manual action): %w", err)
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
	w.Tokens = append(w.Tokens, Token{
		Mint:     newMint,
		Proofs:   newProofs,
		mintedAt: nostr.Now(),
	})

	return nil
}
