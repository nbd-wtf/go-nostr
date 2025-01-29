package nip60

import (
	"context"
	"fmt"
	"time"

	"github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut05"
	"github.com/nbd-wtf/go-nostr/nip60/client"
)

func (w *Wallet) PayBolt11(ctx context.Context, invoice string, opts ...SendOption) (string, error) {
	ss := &sendSettings{}
	for _, opt := range opts {
		opt(ss)
	}

	invoiceAmount, err := getSatoshisAmountFromBolt11(invoice)
	if err != nil {
		return "", err
	}

	w.tokensMu.Lock()
	defer w.tokensMu.Unlock()

	var chosen chosenTokens
	var meltQuote string
	var meltAmount uint64

	invoicePct := uint64(99)
	for range 10 {
		amount := invoiceAmount * invoicePct / 100
		var fee uint64
		chosen, fee, err = w.getProofsForSending(ctx, amount, ss.specificMint)
		if err != nil {
			return "", err
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
			invoicePct--
		} else {
			meltQuote = meltResp.Quote
			meltAmount = meltResp.Amount
			goto meltworked
		}
	}

	return "", fmt.Errorf("stop trying to do the melt because the mint part is too expensive")

meltworked:
	// swap our proofs so we get the exact amount for paying the invoice
	principal, change, err := w.SwapProofs(ctx, chosen.mint, chosen.proofs, meltAmount)
	if err != nil {
		return "", fmt.Errorf("failed to swap at %s into the exact melt amount: %w", chosen.mint, err)
	}

	if err := w.saveChangeAndDeleteUsedTokens(ctx, chosen.mint, change, chosen.tokenIndexes); err != nil {
		return "", err
	}

	// request from mint to _melt_ into paying the invoice
	delay := 200 * time.Millisecond
	// this request will block until the invoice is paid or it fails
	// (but the API also says it can return "pending" so we handle both)
	meltStatus, err := client.PostMeltBolt11(ctx, chosen.mint, nut05.PostMeltBolt11Request{
		Quote:  meltQuote,
		Inputs: principal,
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

	return meltStatus.Preimage, nil
}
