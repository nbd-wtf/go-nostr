package nip11

import (
	"context"
	"encoding/json"
	"fmt"

	"net/http"
	"net/url"
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
	if !strings.HasPrefix(u, "http") && !strings.HasPrefix(u, "ws") {
		u = "wss://" + u
	}
	p, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse url: %s", u)
	}
	if p.Scheme == "ws" {
		p.Scheme = "http"
	} else if p.Scheme == "wss" {
		p.Scheme = "https"
	}
	p.Path = strings.TrimRight(p.Path, "/")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.String(), nil)

	// add the NIP-11 header
	req.Header.Add("Accept", "application/nostr+json")

	// send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	info = &RelayInformationDocument{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(info)
	return info, err
}
