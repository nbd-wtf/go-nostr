package blossom

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/nbd-wtf/go-nostr"
	"github.com/valyala/fasthttp"
)

// httpCall makes an HTTP request to the media server
func (c *Client) httpCall(
	ctx context.Context,
	method string,
	url string,
	contentType string,
	addAuthorization func() string,
	body io.Reader,
	contentSize int64,
	result any,
) error {
	_ = ctx

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(c.mediaserver + url)
	req.Header.SetMethod(method)
	req.Header.SetContentType(contentType)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if addAuthorization != nil {
		auth := addAuthorization()
		if auth != "" {
			req.Header.Add("Authorization", auth)
		}
	}

	if body != nil {
		req.SetBodyStream(body, int(contentSize))
	}

	err := c.httpClient.Do(req, resp)
	if err != nil {
		return fmt.Errorf("failed to call %s: %w\n", url, err)
	}
	if resp.Header.StatusCode() >= 300 {
		reason := resp.Header.Peek("X-Reason")
		return fmt.Errorf("%s returned an error (%d): %s", url, resp.StatusCode(), string(reason))
	}

	if result != nil {
		return json.Unmarshal(resp.Body(), &result)
	}

	return nil
}

// authorizationHeader creates a Nostr-signed authorization header
func (c *Client) authorizationHeader(
	ctx context.Context,
	modify func(*nostr.Event),
) string {
	evt := nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      24242,
		Content:   "blossom stuff",
		Tags: nostr.Tags{
			nostr.Tag{"expiration", strconv.FormatInt(int64(nostr.Now())+60, 10)},
		},
	}

	if modify != nil {
		modify(&evt)
	}

	if err := c.signer.SignEvent(ctx, &evt); err != nil {
		return ""
	}

	jevt, _ := json.Marshal(evt)
	return "Nostr " + base64.StdEncoding.EncodeToString(jevt)
}
