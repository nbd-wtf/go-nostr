package nip60

import (
	"context"
	"fmt"
	"time"

	"github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut05"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip60/client"
)

func (w *Wallet) PayBolt11(ctx context.Context, invoice string, opts ...SendOption) (string, error) {
	if w.PublishUpdate == nil {
		return "", fmt.Errorf("can't do write operations: missing PublishUpdate function")
	}

	ss := &sendSettings{}
	for _, opt := range opts {
		opt(ss)
	}

	invoiceAmount, err := GetSatoshisAmountFromBolt11(invoice)
	if err != nil {
		return "", err
	}

	w.tokensMu.Lock()
	defer w.tokensMu.Unlock()

	var chosen chosenTokens
	var meltQuote string
	var meltAmountWithoutFeeReserve uint64

	feeReservePct := uint64(1)
	feeReserveAbs := uint64(1)

	excludeMints := make([]string, 0, 1)

	for range 5 {
		amount := invoiceAmount*(100+feeReservePct)/100 + feeReserveAbs
		var fee uint64
		chosen, fee, err = w.getProofsForSending(ctx, amount, ss.specificMint, excludeMints)
		if err != nil {
			return "", err
		}

		// we will only do this in mints that support nut08
		if info, _ := client.GetMintInfo(ctx, chosen.mint); info == nil || !info.Nuts.Nut08.Supported {
			excludeMints = append(excludeMints, chosen.mint)
			continue
		}

		// request _melt_ quote (ask the mint how much will it cost to pay a bolt11 invoice)
		meltResp, err := client.PostMeltQuoteBolt11(ctx, chosen.mint, nut05.PostMeltQuoteBolt11Request{
			Request: invoice,
			Unit:    cashu.Sat.String(),
		})
		if err != nil {
			return "", fmt.Errorf("error requesting melt quote from %s: %w", chosen.mint, err)
		}

		// if amount in proofs is not sufficient to pay for the melt request,
		// increase the amount and get proofs again  (because of lighting fees)
		if meltResp.Amount+meltResp.FeeReserve+fee > chosen.proofs.Amount() {
			feeReserveAbs++
		} else {
			meltQuote = meltResp.Quote
			meltAmountWithoutFeeReserve = invoiceAmount + fee
			goto meltworked
		}
	}

	return "", fmt.Errorf("stopped trying to do the melt because all the mints are charging way too much")

meltworked:
	activeKeyset, err := client.GetActiveKeyset(ctx, chosen.mint)
	if err != nil {
		return "", fmt.Errorf("failed to get active keyset for %s: %w", chosen.mint, err)
	}
	ksKeys, err := ParseKeysetKeys(activeKeyset.Keys)
	if err != nil {
		return "", fmt.Errorf("failed to parse keys for %s: %w", chosen.mint, err)
	}

	// since we rely on nut08 we will send all the proofs we've gathered and expect a change
	// we do a split here and discard the principal, as we won't get it back from the mint
	_, preChange, err := splitIntoPrincipalAndChange(
		chosen.keysets,
		chosen.proofs,
		meltAmountWithoutFeeReserve,
		activeKeyset.Id,
		nil,
	)

	// request from mint to _melt_ into paying the invoice
	delay := 200 * time.Millisecond
	// this request will block until the invoice is paid or it fails
	// (but the API also says it can return "pending" so we handle both)
	meltStatus, err := client.PostMeltBolt11(ctx, chosen.mint, nut05.PostMeltBolt11Request{
		Quote:   meltQuote,
		Inputs:  chosen.proofs,
		Outputs: preChange.bm,
	})
inspectmeltstatusresponse:
	if err != nil || meltStatus.State == nut05.Unpaid {
		return "", fmt.Errorf("error melting token: %w", err)
	} else if meltStatus.State == nut05.Unknown {
		return "", fmt.Errorf("we don't know what happened with the melt at %s: %v", chosen.mint, meltStatus)
	} else if meltStatus.State == nut05.Pending {
		for {
			time.Sleep(delay)
			delay *= 2
			meltStatus, err = client.GetMeltQuoteState(ctx, chosen.mint, meltStatus.Quote)
			goto inspectmeltstatusresponse
		}
	}

	// the invoice has been paid, now we save the change we got
	changeProofs, err := constructProofs(preChange, meltStatus.Change, ksKeys)
	if err != nil {
		return "", fmt.Errorf("failed to construct principal proofs: %w", err)
	}

	he := HistoryEntry{
		event:           &nostr.Event{},
		TokenReferences: make([]TokenRef, 0, 5),
		createdAt:       nostr.Now(),
		In:              false,
		Amount:          chosen.proofs.Amount() - changeProofs.Amount(),
	}

	if err := w.saveChangeAndDeleteUsedTokens(ctx, chosen.mint, changeProofs, chosen.tokenIndexes, &he); err != nil {
		return "", err
	}

	w.Lock()
	if err := he.toEvent(ctx, w.kr, he.event); err == nil {
		w.PublishUpdate(*he.event, nil, nil, nil, true)
	}
	w.Unlock()

	return meltStatus.Preimage, nil
}
