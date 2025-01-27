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
	resp, err := httpGet(ctx, mintURL+"/v1/info")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var mintInfo nut06.MintInfo
	if err := json.Unmarshal(body, &mintInfo); err != nil {
		return nil, fmt.Errorf("error reading response from mint: %v", err)
	}

	return &mintInfo, nil
}

func GetActiveKeyset(ctx context.Context, mintURL string) (*nut01.Keyset, error) {
	resp, err := httpGet(ctx, mintURL+"/v1/keys")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var keysetRes nut01.GetKeysResponse
	if err := json.Unmarshal(body, &keysetRes); err != nil {
		return nil, fmt.Errorf("error reading response from mint: %w", err)
	}

	for _, keyset := range keysetRes.Keysets {
		if keyset.Unit == cashu.Sat.String() {
			return &keyset, nil
		}
	}

	return nil, fmt.Errorf("mint has no sat-denominated keyset? %v", keysetRes)
}

func GetAllKeysets(ctx context.Context, mintURL string) ([]nut02.Keyset, error) {
	resp, err := httpGet(ctx, mintURL+"/v1/keysets")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var keysetsRes nut02.GetKeysetsResponse
	if err := json.Unmarshal(body, &keysetsRes); err != nil {
		return nil, fmt.Errorf("error reading response from mint: %v", err)
	}

	return keysetsRes.Keysets, nil
}

func GetKeysetById(ctx context.Context, mintURL, id string) (map[uint64]string, error) {
	resp, err := httpGet(ctx, mintURL+"/v1/keys/"+id)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var keysetRes nut01.GetKeysResponse
	if err := json.Unmarshal(body, &keysetRes); err != nil || len(keysetRes.Keysets) != 1 {
		return nil, fmt.Errorf("error reading response from mint: %w", err)
	}

	return keysetRes.Keysets[0].Keys, nil
}

func PostMintQuoteBolt11(
	ctx context.Context,
	mintURL string,
	mintQuoteRequest nut04.PostMintQuoteBolt11Request,
) (*nut04.PostMintQuoteBolt11Response, error) {
	resp, err := httpPost(ctx, mintURL+"/v1/mint/quote/bolt11", mintQuoteRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var reqMintResponse nut04.PostMintQuoteBolt11Response
	if err := json.Unmarshal(body, &reqMintResponse); err != nil {
		return nil, fmt.Errorf("error reading response from mint: %w", err)
	}

	return &reqMintResponse, nil
}

func GetMintQuoteState(ctx context.Context, mintURL, quoteId string) (*nut04.PostMintQuoteBolt11Response, error) {
	resp, err := httpGet(ctx, mintURL+"/v1/mint/quote/bolt11/"+quoteId)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var mintQuoteResponse nut04.PostMintQuoteBolt11Response
	if err := json.Unmarshal(body, &mintQuoteResponse); err != nil {
		return nil, fmt.Errorf("error reading response from mint: %w", err)
	}

	return &mintQuoteResponse, nil
}

func PostMintBolt11(
	ctx context.Context,
	mintURL string,
	mintRequest nut04.PostMintBolt11Request,
) (*nut04.PostMintBolt11Response, error) {
	resp, err := httpPost(ctx, mintURL+"/v1/mint/bolt11", mintRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var reqMintResponse nut04.PostMintBolt11Response
	if err := json.Unmarshal(body, &reqMintResponse); err != nil {
		return nil, fmt.Errorf("error reading response from mint: %w", err)
	}

	return &reqMintResponse, nil
}

func PostSwap(ctx context.Context, mintURL string, swapRequest nut03.PostSwapRequest) (*nut03.PostSwapResponse, error) {
	resp, err := httpPost(ctx, mintURL+"/v1/swap", swapRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var swapResponse nut03.PostSwapResponse
	if err := json.Unmarshal(body, &swapResponse); err != nil {
		return nil, fmt.Errorf("error reading response from mint: %w", err)
	}

	return &swapResponse, nil
}

func PostMeltQuoteBolt11(
	ctx context.Context,
	mintURL string,
	meltQuoteRequest nut05.PostMeltQuoteBolt11Request,
) (*nut05.PostMeltQuoteBolt11Response, error) {
	resp, err := httpPost(ctx, mintURL+"/v1/melt/quote/bolt11", meltQuoteRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var meltQuoteResponse nut05.PostMeltQuoteBolt11Response
	if err := json.Unmarshal(body, &meltQuoteResponse); err != nil {
		return nil, fmt.Errorf("error reading response from mint: %w", err)
	}

	return &meltQuoteResponse, nil
}

func GetMeltQuoteState(ctx context.Context, mintURL, quoteId string) (*nut05.PostMeltQuoteBolt11Response, error) {
	resp, err := httpGet(ctx, mintURL+"/v1/melt/quote/bolt11/"+quoteId)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var meltQuoteResponse nut05.PostMeltQuoteBolt11Response
	if err := json.Unmarshal(body, &meltQuoteResponse); err != nil {
		return nil, fmt.Errorf("error reading response from mint: %w", err)
	}

	return &meltQuoteResponse, nil
}

func PostMeltBolt11(
	ctx context.Context,
	mintURL string,
	meltRequest nut05.PostMeltBolt11Request,
) (*nut05.PostMeltQuoteBolt11Response, error) {
	resp, err := httpPost(ctx, mintURL+"/v1/melt/bolt11", meltRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var meltResponse nut05.PostMeltQuoteBolt11Response
	if err := json.Unmarshal(body, &meltResponse); err != nil {
		return nil, fmt.Errorf("error reading response from mint: %w", err)
	}

	return &meltResponse, nil
}

func PostCheckProofState(
	ctx context.Context,
	mintURL string,
	stateRequest nut07.PostCheckStateRequest,
) (*nut07.PostCheckStateResponse, error) {
	resp, err := httpPost(ctx, mintURL+"/v1/checkstate", stateRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var stateResponse nut07.PostCheckStateResponse
	if err := json.Unmarshal(body, &stateResponse); err != nil {
		return nil, fmt.Errorf("error reading response from mint: %v", err)
	}

	return &stateResponse, nil
}

func PostRestore(
	ctx context.Context,
	mintURL string,
	restoreRequest nut09.PostRestoreRequest,
) (*nut09.PostRestoreResponse, error) {
	resp, err := httpPost(ctx, mintURL+"/v1/restore", restoreRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var restoreResponse nut09.PostRestoreResponse
	if err := json.Unmarshal(body, &restoreResponse); err != nil {
		return nil, fmt.Errorf("error reading response from mint: %v", err)
	}

	return &restoreResponse, nil
}

func httpGet(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return parse(resp)
}

func httpPost(ctx context.Context, url string, data any) (*http.Response, error) {
	r, w := io.Pipe()
	json.NewEncoder(w).Encode(data)

	req, err := http.NewRequestWithContext(ctx, "POST", url, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return parse(resp)
}

func parse(response *http.Response) (*http.Response, error) {
	if response.StatusCode == 400 {
		var errResponse cashu.Error
		err := json.NewDecoder(response.Body).Decode(&errResponse)
		if err != nil {
			return nil, fmt.Errorf("could not decode error response from mint: %v", err)
		}
		return nil, errResponse
	}

	if response.StatusCode != 200 {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%s", body)
	}

	return response, nil
}
