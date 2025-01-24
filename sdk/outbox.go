package sdk

import (
	"context"
	"strconv"
	"time"
)

var outboxShortTermCache = [256]ostcEntry{}

type ostcEntry struct {
	pubkey string
	relays []string
	when   time.Time
}

func (sys *System) FetchOutboxRelays(ctx context.Context, pubkey string, n int) []string {
	ostcIndex, _ := strconv.ParseUint(pubkey[12:14], 16, 8)
	now := time.Now()
	if entry := outboxShortTermCache[ostcIndex]; entry.pubkey == pubkey && entry.when.Add(time.Minute*2).After(now) {
		return entry.relays
	}

	// if we have it cached that means we have at least tried to fetch recently and it won't be tried again
	fetchGenericList(sys, ctx, pubkey, 10002, kind_10002, parseRelayFromKind10002, sys.RelayListCache)

	relays := sys.Hints.TopN(pubkey, 6)
	if len(relays) == 0 {
		return []string{"wss://relay.damus.io", "wss://nos.lol"}
	}

	// we save a copy of this slice to this cache (must be a copy otherwise
	// we will have a reference to a thing that the caller to this function may change at will)
	relaysCopy := make([]string, len(relays))
	copy(relaysCopy, relays)
	outboxShortTermCache[ostcIndex] = ostcEntry{pubkey, relaysCopy, now}

	if len(relays) > n {
		relays = relays[0:n]
	}

	return relays
}

func (sys *System) FetchInboxRelays(ctx context.Context, pubkey string, n int) []string {
	rl := sys.FetchRelayList(ctx, pubkey)
	if len(rl.Items) == 0 || len(rl.Items) > 7 {
		return []string{"wss://relay.damus.io", "wss://nos.lol"}
	}

	relays := make([]string, 0, n)
	for _, r := range rl.Items {
		if r.Inbox {
			relays = append(relays, r.URL)
		}
	}

	return relays
}
