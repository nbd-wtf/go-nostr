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
	"github.com/elnosh/gonuts/cashu/nuts/nut13"
	"github.com/nbd-wtf/go-nostr/nip60/client"
)

type swapSettings struct {
	spendingCondition *nut10.SpendingCondition
	mustSignOutputs   bool
}

func (w *Wallet) swapProofs(
	ctx context.Context,
	mint string,
	proofs cashu.Proofs,
	targetAmount uint64,
	ss swapSettings,
) (principal cashu.Proofs, change cashu.Proofs, err error) {
	keysetIdList := []string{}
	for i := range w.Mints {
		if w.Mints[i] != mint {
			keysets, err := client.GetAllKeysets(ctx, w.Mints[i])
			if err != nil {
				return nil, nil, fmt.Errorf("could not get keysets for all previous keysets %s: %w", w.Mints[i], err)
			}

			for j := range keysets {
				keysetIdList = append(keysetIdList, keysets[j].Id)
			}
		}
	}

	keysets, err := client.GetAllKeysets(ctx, mint)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get all keysets for %s: %w", mint, err)
	}

	for j := range keysets {
		err := nut13.CheckCollidingKeysets(keysetIdList, []string{keysets[j].Id})
		if err != nil {
			return nil, nil, fmt.Errorf("encountered keyset collition for mint %s: %w", mint, err)
		}
		keysetIdList = append(keysetIdList, keysets[j].Id)
	}

	activeKeyset, err := client.GetActiveKeyset(ctx, mint)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get active keyset for %s: %w", mint, err)
	}
	ksKeys, err := ParseKeysetKeys(activeKeyset.Keys)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse keys for %s: %w", mint, err)
	}

	prePrincipal, preChange, err := splitIntoPrincipalAndChange(
		keysets,
		proofs,
		targetAmount,
		activeKeyset.Id,
		ss.spendingCondition,
	)
	if err != nil {
		return nil, nil, err
	}

	if ss.mustSignOutputs {
		for i, output := range prePrincipal.bm {
			prePrincipal.bm[i].Witness, err = signOutput(w.PrivateKey, output)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to sign output message %d: %w", i, err)
			}
		}
	}

	req := nut03.PostSwapRequest{
		Inputs:  proofs,
		Outputs: slices.Concat(prePrincipal.bm, preChange.bm),
	}

	res, err := client.PostSwap(ctx, mint, req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to swap tokens at %s: %w", mint, err)
	}

	// build the proofs locally from mint's response
	principal, err = constructProofs(prePrincipal, res.Signatures[0:len(prePrincipal.bm)], ksKeys)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to construct principal proofs: %w", err)
	}

	change, err = constructProofs(preChange, res.Signatures[len(prePrincipal.bm):], ksKeys)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to construct principal proofs: %w", err)
	}

	return principal, change, nil
}

type preparedOutputs struct {
	bm      cashu.BlindedMessages
	rs      []*btcec.PrivateKey
	secrets []string
}

func splitIntoPrincipalAndChange(
	keysets []nut02.Keyset,
	proofs cashu.Proofs,
	targetAmount uint64,
	activeKeysetId string,
	spendingCondition *nut10.SpendingCondition,
) (principal preparedOutputs, change preparedOutputs, err error) {
	// decide the shape of the proofs we'll swap for
	proofsAmount := proofs.Amount()
	var (
		principalAmount uint64
		changeAmount    uint64
	)
	fee := calculateFee(proofs, keysets)
	if targetAmount < proofsAmount {
		// we'll get the exact target, then a change, and fee will be taken from the change
		principalAmount = targetAmount
		changeAmount = proofsAmount - targetAmount - fee
	} else if targetAmount == proofsAmount {
		// we're swapping everything, so take the fee from the principal
		principalAmount = targetAmount - fee
		changeAmount = 0
	} else {
		err = fmt.Errorf("can't swap for more than we are sending: %d > %d", targetAmount, proofsAmount)
		return
	}
	splits := make([]uint64, 0, len(proofs)*2)
	splits = append(splits, cashu.AmountSplit(principalAmount)...)
	changeStartIndex := len(splits)
	splits = append(splits, cashu.AmountSplit(changeAmount)...)

	// prepare message to send to mint
	bm, secrets, rs, err := createBlindedMessages(splits, activeKeysetId, spendingCondition)
	if err != nil {
		err = fmt.Errorf("failed to create blinded message: %w", err)
		return
	}

	return preparedOutputs{
			bm:      bm[0:changeStartIndex],
			rs:      rs[0:changeStartIndex],
			secrets: secrets[0:changeStartIndex],
		}, preparedOutputs{
			bm:      bm[changeStartIndex:],
			rs:      rs[changeStartIndex:],
			secrets: secrets[changeStartIndex:],
		}, nil
}
