package nip05

import (
	"context"
	"encoding/hex"
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
	spl := strings.Split(fullname, "@")

	var name, domain string
	switch len(spl) {
	case 1:
		name = "_"
		domain = spl[0]
	case 2:
		name = spl[0]
		domain = spl[1]
	default:
		return nil, fmt.Errorf("not a valid nip-05 identifier")
	}

	if strings.Index(domain, ".") == -1 {
		return nil, fmt.Errorf("hostname doesn't have a dot")
	}

	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("https://%s/.well-known/nostr.json?name=%s", domain, name), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create a request: %w", err)
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	var result WellKnownResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode json response: %w", err)
	}

	pubkey, ok := result.Names[name]
	if !ok {
		return &nostr.ProfilePointer{}, nil
	}

	if len(pubkey) == 64 {
		if _, err := hex.DecodeString(pubkey); err != nil {
			return &nostr.ProfilePointer{}, nil
		}
	}

	relays, _ := result.Relays[pubkey]

	return &nostr.ProfilePointer{
		PublicKey: pubkey,
		Relays:    relays,
	}, nil
}

func NormalizeIdentifier(fullname string) string {
	if strings.HasPrefix(fullname, "_@") {
		return fullname[2:]
	}

	return fullname
}
