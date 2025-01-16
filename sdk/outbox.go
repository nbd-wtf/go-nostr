package sdk

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
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

func (sys *System) ExpandQueriesByAuthorAndRelays(
	ctx context.Context,
	filter nostr.Filter,
) (map[string]nostr.Filter, error) {
	n := len(filter.Authors)
	if n == 0 {
		return nil, fmt.Errorf("no authors in filter")
	}

	relaysForPubkey := make(map[string][]string, n)
	mu := sync.Mutex{}

	wg := sync.WaitGroup{}
	wg.Add(n)
	for _, pubkey := range filter.Authors {
		go func(pubkey string) {
			defer wg.Done()
			relayURLs := sys.FetchOutboxRelays(ctx, pubkey, 3)
			c := 0
			for _, r := range relayURLs {
				relay, err := sys.Pool.EnsureRelay(r)
				if err != nil {
					continue
				}
				mu.Lock()
				relaysForPubkey[pubkey] = append(relaysForPubkey[pubkey], relay.URL)
				mu.Unlock()
				c++
				if c == 3 {
					return
				}
			}
		}(pubkey)
	}
	wg.Wait()

	filterForRelay := make(map[string]nostr.Filter, n) // { [relay]: filter }
	for pubkey, relays := range relaysForPubkey {
		for _, relay := range relays {
			flt, ok := filterForRelay[relay]
			if !ok {
				flt = filter.Clone()
				filterForRelay[relay] = flt
			}
			flt.Authors = append(flt.Authors, pubkey)
		}
	}

	return filterForRelay, nil
}
