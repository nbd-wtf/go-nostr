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

type (
	name2KeyMap   map[string]string
	key2RelaysMap map[string][]string
)

type WellKnownResponse struct {
	Names  name2KeyMap   `json:"names"`  // NIP-05
	Relays key2RelaysMap `json:"relays"` // NIP-35
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

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://%s/.well-known/nostr.json?name=%s", domain, name), nil)
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

	var result WellKnownResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode json response: %w", err)
	}

	pubkey, ok := result.Names[name]
	if !ok {
		return nil, nil
	}

	if len(pubkey) == 64 {
		if _, err := hex.DecodeString(pubkey); err != nil {
			return nil, nil
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
