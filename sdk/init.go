package sdk

import (
	"context"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/sdk/cache"
)

type System struct {
	relaysCache      cache.Cache32[[]Relay]
	followsCache     cache.Cache32[[]Follow]
	metadataCache    cache.Cache32[ProfileMetadata]
	pool             *nostr.SimplePool
	relayListRelays  []string
	followListRelays []string
	metadataRelays   []string
}

func (sys System) FetchRelays(ctx context.Context, pubkey string) []Relay {
	if v, ok := sys.relaysCache.Get(pubkey); ok {
		return v
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	res := FetchRelaysForPubkey(ctx, sys.pool, pubkey, sys.relayListRelays...)
	sys.relaysCache.SetWithTTL(pubkey, res, time.Hour*6)
	return res
}

func (sys System) FetchOutboxRelays(ctx context.Context, pubkey string) []string {
	relays := sys.FetchRelays(ctx, pubkey)
	result := make([]string, 0, len(relays))
	for _, relay := range relays {
		if relay.Outbox {
			result = append(result, relay.URL)
		}
	}
	return result
}

func (sys System) FetchProfileMetadata(ctx context.Context, pubkey string) ProfileMetadata {
	if v, ok := sys.metadataCache.Get(pubkey); ok {
		return v
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	res := FetchProfileMetadata(ctx, sys.pool, pubkey, sys.metadataRelays...)
	sys.metadataCache.SetWithTTL(pubkey, res, time.Hour*6)
	return res
}
