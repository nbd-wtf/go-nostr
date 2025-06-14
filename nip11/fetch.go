package nip11

import (
	"context"
	"fmt"
	"net/http"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/nbd-wtf/go-nostr"
)

// Fetch fetches the NIP-11 metadata for a relay.
//
// It will always return `info` with at least `URL` filled -- even if we can't connect to the
// relay or if it doesn't have a NIP-11 handler -- although in that case it will also return
// an error.
func Fetch(ctx context.Context, u string) (info RelayInformationDocument, err error) {
	// normalize URL to start with http://, https:// or without protocol
	u = nostr.NormalizeURL(u)
	if len(u) < 8 {
		return info, fmt.Errorf("invalid url %s", u)
	}

	info = RelayInformationDocument{
		URL: u,
	}

	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 7*time.Second)
		defer cancel()
	}

	// make request
	req, _ := http.NewRequestWithContext(ctx, "GET", "http"+u[2:], nil)

	// add the NIP-11 headers
	req.Header.Add("Accept", "application/nostr+json")
	req.Header.Add("User-Agent", "https://github.com/nbd-wtf/go-nostr")

	// send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return info, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := jsoniter.NewDecoder(resp.Body).Decode(&info); err != nil {
		return info, fmt.Errorf("invalid json: %w", err)
	}

	return info, nil
}
