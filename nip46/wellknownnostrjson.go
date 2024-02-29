package nip46

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr/nip05"
)

func queryWellKnownNostrJson(ctx context.Context, fullname string) (pubkey string, relays []string, err error) {
	result, name, err := nip05.Fetch(ctx, fullname)
	if err != nil {
		return "", nil, err
	}

	pubkey, ok := result.Names[name]
	if !ok {
		return "", nil, fmt.Errorf("no entry found for the '%s' name", name)
	}
	relays, _ = result.NIP46[pubkey]
	if !ok {
		return "", nil, fmt.Errorf("no bunker relays found for the '%s' name", name)
	}

	return pubkey, relays, nil
}
