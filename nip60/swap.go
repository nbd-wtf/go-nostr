package nip60

import (
	"context"
	"fmt"
	"time"

	"github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut02"
	"github.com/elnosh/gonuts/cashu/nuts/nut04"
	"github.com/elnosh/gonuts/cashu/nuts/nut05"
	"github.com/nbd-wtf/go-nostr/nip60/client"
)

// lightningMeltMint does the lightning dance of moving funds between mints
func lightningMeltMint(
	ctx context.Context,
	proofs cashu.Proofs,
	from string,
	fromKeysets []nut02.Keyset,
	to string,
) (newProofs cashu.Proofs, err error, canTryWithAnotherTargetMint bool, manualActionRequired bool) {
	// get active keyset of target mint
	keyset, err := client.GetActiveKeyset(ctx, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyset keys for %s: %w", to, err), true, false
	}

	// unblind the signatures from the promises and build the proofs
	keysetKeys, err := parseKeysetKeys(keyset.Keys)
	if err != nil {
		return nil, fmt.Errorf("target mint %s sent us an invalid keyset: %w", to, err), true, false
	}

	// now we start the melt-mint process in multiple attempts
	invoicePct := 0.99
	proofsAmount := proofs.Amount()
	amount := float64(proofsAmount) * invoicePct
	fee := uint64(calculateFee(proofs, fromKeysets))
	var meltQuote string
	var mintQuote string
	for range 10 {
		// request _mint_ quote to the 'to' mint -- this will generate an invoice
		mintResp, err := client.PostMintQuoteBolt11(ctx, to, nut04.PostMintQuoteBolt11Request{
			Amount: uint64(amount) - fee,
			Unit:   cashu.Sat.String(),
		})
		if err != nil {
			return nil, fmt.Errorf("error requesting mint quote from %s: %w", to, err), true, false
		}

		// request _melt_ quote from the 'from' mint
		// this melt will pay the invoice generated from the previous mint quote request
		meltResp, err := client.PostMeltQuoteBolt11(ctx, from, nut05.PostMeltQuoteBolt11Request{
			Request: mintResp.Request,
			Unit:    cashu.Sat.String(),
		})
		if err != nil {
			return nil, fmt.Errorf("error requesting melt quote from %s: %w", from, err), false, false
		}

		// if amount in proofs is less than amount asked from mint in melt request,
		// lower the amount for mint request (because of lighting fees?)
		if meltResp.Amount+meltResp.FeeReserve+fee > proofsAmount {
			invoicePct -= 0.01
			amount *= invoicePct
		} else {
			meltQuote = meltResp.Quote
			mintQuote = mintResp.Quote
			goto meltworked
		}
	}

	return nil, fmt.Errorf("stop trying to do the melt because the mint part is too expensive"), true, false

meltworked:
	// request from mint to pay invoice from the mint quote request
	_, err = client.PostMeltBolt11(ctx, from, nut05.PostMeltBolt11Request{
		Quote:  meltQuote,
		Inputs: proofs,
	})
	if err != nil {
		return nil, fmt.Errorf("error melting token: %v", err), false, true
	}

	sleepTime := time.Millisecond * 200
	failures := 0
	for range 12 {
		sleepTime *= 2
		time.Sleep(sleepTime)

		// check if the _mint_ invoice was paid
		mintQuoteStatusResp, err := client.GetMintQuoteState(ctx, to, mintQuote)
		if err != nil {
			failures++
			if failures > 10 {
				return nil, fmt.Errorf(
					"target mint %s failed to answer to our mint quote checks (%s): %w; a manual fix is needed",
					to, meltQuote, err,
				), false, true
			}
		}

		// if it wasn't paid try again
		if mintQuoteStatusResp.State != nut04.Paid {
			continue
		}

		// if it got paid make proceed to get proofs
		split := []uint64{1, 2, 3, 4}
		blindedMessages, secrets, rs, err := createBlindedMessages(split, keyset.Id)
		if err != nil {
			return nil, fmt.Errorf("error creating blinded messages: %v", err), false, true
		}

		// request mint to sign the blinded messages
		mintResponse, err := client.PostMintBolt11(ctx, to, nut04.PostMintBolt11Request{
			Quote:   mintQuote,
			Outputs: blindedMessages,
		})
		if err != nil {
			return nil, fmt.Errorf("mint request to %s failed (%s): %w", to, mintQuote, err), false, true
		}

		proofs, err := constructProofs(mintResponse.Signatures, blindedMessages, secrets, rs, keysetKeys)
		if err != nil {
			return nil, fmt.Errorf("error constructing proofs: %w", err), false, true
		}

		return proofs, nil, false, false
	}

	return nil, fmt.Errorf("we gave up waiting for the invoice at %s to be paid: %s", to, meltQuote), false, true
}
