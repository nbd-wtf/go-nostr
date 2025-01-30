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

type lightningSwapStatus int

const (
	nothingCanBeDone = iota
	tryAnotherTargetMint
	storeTokenFromSourceMint
	manualActionRequired
)

// lightningMeltMint does the lightning dance of moving funds between mints
func lightningMeltMint(
	ctx context.Context,
	proofs cashu.Proofs,
	from string,
	fromKeysets []nut02.Keyset,
	to string,
) (cashu.Proofs, error, lightningSwapStatus) {
	// get active keyset of target mint
	keyset, err := client.GetActiveKeyset(ctx, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyset keys for %s: %w", to, err), tryAnotherTargetMint
	}

	// unblind the signatures from the promises and build the proofs
	keysetKeys, err := parseKeysetKeys(keyset.Keys)
	if err != nil {
		return nil, fmt.Errorf("target mint %s sent us an invalid keyset: %w", to, err), tryAnotherTargetMint
	}

	// now we start the melt-mint process in multiple attempts
	invoicePct := uint64(99)
	proofsAmount := proofs.Amount()
	amount := proofsAmount * invoicePct / 100
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
			return nil, fmt.Errorf("error requesting mint quote from %s: %w", to, err), tryAnotherTargetMint
		}

		// request _melt_ quote from the 'from' mint
		// this melt will pay the invoice generated from the previous mint quote request
		meltResp, err := client.PostMeltQuoteBolt11(ctx, from, nut05.PostMeltQuoteBolt11Request{
			Request: mintResp.Request,
			Unit:    cashu.Sat.String(),
		})
		if err != nil {
			return nil, fmt.Errorf("error requesting melt quote from %s: %w", from, err), storeTokenFromSourceMint
		}

		// if amount in proofs is less than amount asked from mint in melt request,
		// lower the amount for mint request (because of lighting fees)
		if meltResp.Amount+meltResp.FeeReserve+fee > proofsAmount {
			invoicePct--
			amount = proofsAmount * invoicePct / 100
		} else {
			meltQuote = meltResp.Quote
			mintQuote = mintResp.Quote
			goto meltworked
		}
	}

	return nil, fmt.Errorf("stop trying to do the melt because the mint part is too expensive"), tryAnotherTargetMint

meltworked:
	// request from mint to _melt_ into paying the invoice
	delay := 200 * time.Millisecond
	// this request will block until the invoice is paid or it fails
	// (but the API also says it can return "pending" so we handle both)
	meltStatus, err := client.PostMeltBolt11(ctx, from, nut05.PostMeltBolt11Request{
		Quote:  meltQuote,
		Inputs: proofs,
	})
inspectmeltstatusresponse:
	if err != nil || meltStatus.State == nut05.Unpaid {
		return nil, fmt.Errorf("error melting token: %w", err), storeTokenFromSourceMint
	} else if meltStatus.State == nut05.Unknown {
		return nil,
			fmt.Errorf("we don't know what happened with the melt at %s: %v", from, meltStatus),
			manualActionRequired
	} else if meltStatus.State == nut05.Pending {
		for {
			time.Sleep(delay)
			delay *= 2
			meltStatus, err = client.GetMeltQuoteState(ctx, from, meltStatus.Quote)
			goto inspectmeltstatusresponse
		}
	}

	// source mint says it has paid the invoice, now check it against the target mint
	// check if the _mint_ invoice was paid
	mintQuoteStatusResp, err := client.GetMintQuoteState(ctx, to, mintQuote)
	if err != nil {
		return nil, fmt.Errorf(
			"target mint %s failed to answer to our mint quote checks (%s): %w; a manual fix is needed",
			to, meltQuote, err,
		), manualActionRequired
	}
	if mintQuoteStatusResp.State != nut04.Paid {
		return nil, fmt.Errorf(
			"target mint %s says the invoice wasn't paid although the source mint %s said it did, %s -> %s",
			to, from, meltQuote, mintQuote,
		), manualActionRequired
	}

	// if it got paid make proceed to get proofs
	split := cashu.AmountSplit(amount)
	blindedMessages, secrets, rs, err := createBlindedMessages(split, keyset.Id, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating blinded messages: %v", err), manualActionRequired
	}

	// request mint to sign the blinded messages
	mintResponse, err := client.PostMintBolt11(ctx, to, nut04.PostMintBolt11Request{
		Quote:   mintQuote,
		Outputs: blindedMessages,
	})
	if err != nil {
		return nil, fmt.Errorf("mint request to %s failed (%s): %w", to, mintQuote, err), manualActionRequired
	}

	proofs, err = constructProofs(preparedOutputs{
		bm:      blindedMessages,
		secrets: secrets,
		rs:      rs,
	}, mintResponse.Signatures, keysetKeys)
	if err != nil {
		return nil, fmt.Errorf("error constructing proofs: %w", err), manualActionRequired
	}

	return proofs, nil, nothingCanBeDone
}
