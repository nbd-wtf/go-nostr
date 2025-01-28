package nip60

import (
	"context"
	"fmt"

	"github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut03"
	"github.com/elnosh/gonuts/cashu/nuts/nut10"
	"github.com/nbd-wtf/go-nostr/nip60/client"
)

type SwapOption func(*swapSettings)

func WithSignedOutputs() SwapOption {
	return func(ss *swapSettings) {
		ss.mustSignOutputs = true
	}
}

func WithSpendingCondition(sc nut10.SpendingCondition) SwapOption {
	return func(ss *swapSettings) {
		ss.spendingCondition = &sc
	}
}

type swapSettings struct {
	spendingCondition *nut10.SpendingCondition
	mustSignOutputs   bool
}

func (w *Wallet) SwapProofs(
	ctx context.Context,
	mint string,
	proofs cashu.Proofs,
	targetAmount uint64,
	opts ...SwapOption,
) (principal cashu.Proofs, change cashu.Proofs, err error) {
	var ss swapSettings
	for _, opt := range opts {
		opt(&ss)
	}

	// fetch all this keyset drama first
	keysets, err := client.GetAllKeysets(ctx, mint)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get all keysets for %s: %w", mint, err)
	}
	activeKeyset, err := client.GetActiveKeyset(ctx, mint)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get active keyset for %s: %w", mint, err)
	}
	ksKeys, err := parseKeysetKeys(activeKeyset.Keys)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse keys for %s: %w", mint, err)
	}

	// decide the shape of the proofs we'll swap for
	proofsAmount := proofs.Amount()
	var (
		principalAmount uint64
		changeAmount    uint64
	)
	fee := calculateFee(proofs, keysets)
	if targetAmount < proofsAmount {
		// we'll get the exact target, then a change, and fee will be taken from the change
		changeAmount = proofsAmount - targetAmount - fee
	} else if targetAmount == proofsAmount {
		// we're swapping everything, so take the fee from the principal
		principalAmount = targetAmount - fee
	} else {
		return nil, nil, fmt.Errorf("can't swap for more than we are sending: %d > %d",
			targetAmount, proofsAmount)
	}
	splits := make([]uint64, 0, len(proofs)*2)
	splits = append(splits, cashu.AmountSplit(principalAmount)...)
	changeStartIndex := len(splits)
	splits = append(splits, cashu.AmountSplit(changeAmount)...)

	// prepare message to send to mint
	outputs, secrets, rs, err := createBlindedMessages(splits, activeKeyset.Id, ss.spendingCondition)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create blinded message: %w", err)
	}

	if ss.mustSignOutputs {
		for i, output := range outputs {
			outputs[i].Witness, err = signOutput(w.PrivateKey, output)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to sign output message %d: %w", i, err)
			}
		}
	}

	req := nut03.PostSwapRequest{
		Inputs:  proofs,
		Outputs: outputs,
	}

	res, err := client.PostSwap(ctx, mint, req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to claim received tokens at %s: %w", mint, err)
	}

	// build the proofs locally from mint's response
	newProofs, err := constructProofs(res.Signatures, req.Outputs, secrets, rs, ksKeys)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to construct proofs: %w", err)
	}

	return newProofs[0:changeStartIndex], newProofs[changeStartIndex:], nil
}
