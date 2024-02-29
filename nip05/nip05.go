package nip05

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

type WellKnownResponse struct {
	Names  map[string]string   `json:"names"`
	Relays map[string][]string `json:"relays,omitempty"`
	NIP46  map[string][]string `json:"nip46,omitempty"`
}

func QueryIdentifier(ctx context.Context, fullname string) (*nostr.ProfilePointer, error) {
	result, name, err := Fetch(ctx, fullname)
	if err != nil {
		return nil, err
	}

	pubkey, ok := result.Names[name]
	if !ok {
		return nil, fmt.Errorf("no entry for name '%s'", name)
	}

	if !nostr.IsValidPublicKey(pubkey) {
		return nil, fmt.Errorf("got an invalid public key '%s'", pubkey)
	}

	relays, _ := result.Relays[pubkey]
	return &nostr.ProfilePointer{
		PublicKey: pubkey,
		Relays:    relays,
	}, nil
}

func Fetch(ctx context.Context, fullname string) (resp WellKnownResponse, name string, err error) {
	spl := strings.Split(fullname, "@")

	var domain string
	switch len(spl) {
	case 1:
		name = "_"
		domain = spl[0]
	case 2:
		name = spl[0]
		domain = spl[1]
	default:
		return resp, name, fmt.Errorf("not a valid nip-05 identifier")
	}

	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("https://%s/.well-known/nostr.json?name=%s", domain, name), nil)
	if err != nil {
		return resp, name, fmt.Errorf("failed to create a request: %w", err)
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	res, err := client.Do(req)
	if err != nil {
		return resp, name, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	var result WellKnownResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return resp, name, fmt.Errorf("failed to decode json response: %w", err)
	}

	return resp, name, nil
}

func NormalizeIdentifier(fullname string) string {
	if strings.HasPrefix(fullname, "_@") {
		return fullname[2:]
	}

	return fullname
}
