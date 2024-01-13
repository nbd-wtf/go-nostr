package nip11

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Fetch fetches the NIP-11 RelayInformationDocument.
func Fetch(ctx context.Context, u string) (info *RelayInformationDocument, err error) {
	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 7*time.Second)
		defer cancel()
	}

	// normalize URL to start with http:// or https://
	if strings.HasPrefix(u, "ws") {
		u = "http" + u[2:]
	}
	u = strings.TrimRight(u, "/")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)

	// add the NIP-11 header
	req.Header.Add("Accept", "application/nostr+json")

	// send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	info = &RelayInformationDocument{}
	if err := json.NewDecoder(resp.Body).Decode(info); err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}

	return info, nil
}
