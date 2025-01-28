package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut01"
	"github.com/elnosh/gonuts/cashu/nuts/nut02"
	"github.com/elnosh/gonuts/cashu/nuts/nut03"
	"github.com/elnosh/gonuts/cashu/nuts/nut04"
	"github.com/elnosh/gonuts/cashu/nuts/nut05"
	"github.com/elnosh/gonuts/cashu/nuts/nut06"
	"github.com/elnosh/gonuts/cashu/nuts/nut07"
	"github.com/elnosh/gonuts/cashu/nuts/nut09"
)

func GetMintInfo(ctx context.Context, mintURL string) (*nut06.MintInfo, error) {
	var mintInfo nut06.MintInfo
	if err := httpGet(ctx, mintURL+"/v1/info", &mintInfo); err != nil {
		return nil, err
	}
	return &mintInfo, nil
}

func GetActiveKeyset(ctx context.Context, mintURL string) (*nut01.Keyset, error) {
	var keysetRes nut01.GetKeysResponse
	if err := httpGet(ctx, mintURL+"/v1/keys", &keysetRes); err != nil {
		return nil, err
	}

	for _, keyset := range keysetRes.Keysets {
		if keyset.Unit == cashu.Sat.String() {
			return &keyset, nil
		}
	}

	return nil, fmt.Errorf("mint has no sat-denominated keyset? %v", keysetRes)
}

func GetAllKeysets(ctx context.Context, mintURL string) ([]nut02.Keyset, error) {
	var keysetsRes nut02.GetKeysetsResponse
	if err := httpGet(ctx, mintURL+"/v1/keysets", &keysetsRes); err != nil {
		return nil, err
	}
	return keysetsRes.Keysets, nil
}

func GetKeysetById(ctx context.Context, mintURL, id string) (map[uint64]string, error) {
	var keysetRes nut01.GetKeysResponse
	if err := httpGet(ctx, mintURL+"/v1/keys/"+id, &keysetRes); err != nil {
		return nil, err
	}
	return keysetRes.Keysets[0].Keys, nil
}

func PostMintQuoteBolt11(
	ctx context.Context,
	mintURL string,
	mintQuoteRequest nut04.PostMintQuoteBolt11Request,
) (*nut04.PostMintQuoteBolt11Response, error) {
	var reqMintResponse nut04.PostMintQuoteBolt11Response
	if err := httpPost(ctx, mintURL+"/v1/mint/quote/bolt11", mintQuoteRequest, &reqMintResponse); err != nil {
		return nil, err
	}
	return &reqMintResponse, nil
}

func GetMintQuoteState(ctx context.Context, mintURL, quoteId string) (*nut04.PostMintQuoteBolt11Response, error) {
	var mintQuoteResponse nut04.PostMintQuoteBolt11Response
	if err := httpGet(ctx, mintURL+"/v1/mint/quote/bolt11/"+quoteId, &mintQuoteResponse); err != nil {
		return nil, err
	}
	return &mintQuoteResponse, nil
}

func PostMintBolt11(
	ctx context.Context,
	mintURL string,
	mintRequest nut04.PostMintBolt11Request,
) (*nut04.PostMintBolt11Response, error) {
	var reqMintResponse nut04.PostMintBolt11Response
	if err := httpPost(ctx, mintURL+"/v1/mint/bolt11", mintRequest, &reqMintResponse); err != nil {
		return nil, err
	}
	return &reqMintResponse, nil
}

func PostSwap(ctx context.Context, mintURL string, swapRequest nut03.PostSwapRequest) (*nut03.PostSwapResponse, error) {
	var swapResponse nut03.PostSwapResponse
	if err := httpPost(ctx, mintURL+"/v1/swap", swapRequest, &swapResponse); err != nil {
		return nil, err
	}
	return &swapResponse, nil
}

func PostMeltQuoteBolt11(
	ctx context.Context,
	mintURL string,
	meltQuoteRequest nut05.PostMeltQuoteBolt11Request,
) (*nut05.PostMeltQuoteBolt11Response, error) {
	var meltQuoteResponse nut05.PostMeltQuoteBolt11Response
	if err := httpPost(ctx, mintURL+"/v1/melt/quote/bolt11", meltQuoteRequest, &meltQuoteResponse); err != nil {
		return nil, err
	}
	return &meltQuoteResponse, nil
}

func GetMeltQuoteState(ctx context.Context, mintURL, quoteId string) (*nut05.PostMeltQuoteBolt11Response, error) {
	var meltQuoteResponse nut05.PostMeltQuoteBolt11Response
	if err := httpGet(ctx, mintURL+"/v1/melt/quote/bolt11/"+quoteId, &meltQuoteResponse); err != nil {
		return nil, err
	}
	return &meltQuoteResponse, nil
}

func PostMeltBolt11(
	ctx context.Context,
	mintURL string,
	meltRequest nut05.PostMeltBolt11Request,
) (*nut05.PostMeltQuoteBolt11Response, error) {
	var meltResponse nut05.PostMeltQuoteBolt11Response
	if err := httpPost(ctx, mintURL+"/v1/melt/bolt11", meltRequest, &meltResponse); err != nil {
		return nil, err
	}
	return &meltResponse, nil
}

func PostCheckProofState(
	ctx context.Context,
	mintURL string,
	stateRequest nut07.PostCheckStateRequest,
) (*nut07.PostCheckStateResponse, error) {
	var stateResponse nut07.PostCheckStateResponse
	if err := httpPost(ctx, mintURL+"/v1/checkstate", stateRequest, &stateResponse); err != nil {
		return nil, err
	}
	return &stateResponse, nil
}

func PostRestore(
	ctx context.Context,
	mintURL string,
	restoreRequest nut09.PostRestoreRequest,
) (*nut09.PostRestoreResponse, error) {
	var restoreResponse nut09.PostRestoreResponse
	if err := httpPost(ctx, mintURL+"/v1/restore", restoreRequest, &restoreResponse); err != nil {
		return nil, err
	}
	return &restoreResponse, nil
}

func httpGet(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	return parse(resp, dst)
}

func httpPost(ctx context.Context, url string, data any, dst any) error {
	r, w := io.Pipe()
	go func() {
		json.NewEncoder(w).Encode(data)
		w.Close()
	}()

	req, err := http.NewRequestWithContext(ctx, "POST", url, r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return parse(resp, dst)
}

func parse(response *http.Response, dst any) error {
	if response.StatusCode == 400 {
		var errResponse cashu.Error
		err := json.NewDecoder(response.Body).Decode(&errResponse)
		if err != nil {
			return fmt.Errorf("could not decode error response from mint: %v", err)
		}
		return errResponse
	}

	if response.StatusCode != 200 {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", body)
	}

	err := json.NewDecoder(response.Body).Decode(dst)
	if err != nil {
		return fmt.Errorf("could not decode response from mint: %w", err)
	}

	return nil
}
