package sdk

import (
	"context"
	"encoding/hex"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/nbd-wtf/go-nostr"
)

const (
	pubkeyStreamLatestPrefix = byte('L')
	pubkeyStreamOldestPrefix = byte('O')
)

func makePubkeyStreamKey(prefix byte, pubkey string) []byte {
	key := make([]byte, 1+8)
	key[0] = prefix
	hex.Decode(key[1:], []byte(pubkey[0:16]))
	return key
}

// StreamPubkeysForward starts listening for new events from the given pubkeys,
// taking into account their outbox relays. It returns a channel that emits events
// continuously. The events are fetched from the time of the last seen event for
// each pubkey (stored in KVStore) onwards.
func (sys *System) StreamLiveFeed(
	ctx context.Context,
	pubkeys []string,
	kinds []int,
) (<-chan *nostr.Event, error) {
	events := make(chan *nostr.Event)

	active := atomic.Int32{}
	active.Add(int32(len(pubkeys)))

	// start a subscription for each relay group
	for _, pubkey := range pubkeys {
		relays := sys.FetchOutboxRelays(ctx, pubkey, 2)
		if len(relays) == 0 {
			if active.Add(-1) == 0 {
				close(events)
			}
			continue
		}

		latestKey := makePubkeyStreamKey(pubkeyStreamLatestPrefix, pubkey)
		latest := nostr.Timestamp(0)
		oldestKey := makePubkeyStreamKey(pubkeyStreamOldestPrefix, pubkey)
		oldest := nostr.Timestamp(0)

		serial := 0

		var since *nostr.Timestamp
		if data, _ := sys.KVStore.Get(latestKey); data != nil {
			latest = decodeTimestamp(data)
			since = &latest
		}

		filter := nostr.Filter{
			Authors: []string{pubkey},
			Since:   since,
			Kinds:   kinds,
		}

		go func() {
			sub := sys.Pool.SubMany(ctx, relays, nostr.Filters{filter})
			for evt := range sub {
				sys.StoreRelay.Publish(ctx, *evt.Event)
				if latest < evt.CreatedAt {
					latest = evt.CreatedAt
					serial++
					if serial%10 == 0 {
						sys.KVStore.Set(latestKey, encodeTimestamp(latest))
					}
				} else if oldest > evt.CreatedAt {
					oldest = evt.CreatedAt
					sys.KVStore.Set(oldestKey, encodeTimestamp(oldest))
				}

				events <- evt.Event
			}

			if active.Add(-1) == 0 {
				close(events)
			}
		}()
	}

	return events, nil
}

// FetchFeedNextPage fetches historical events from the given pubkeys in descending order starting from the
// given until timestamp. The limit argument is just a hint of how much content you want for the entire list,
// it isn't guaranteed that this quantity of events will be returned -- it could be more or less.
//
// It relies on KVStore's latestKey and oldestKey in order to determine if we should go to relays to ask
// for events or if we should just return what we have stored locally.
func (sys *System) FetchFeedPage(
	ctx context.Context,
	pubkeys []string,
	kinds []int,
	until nostr.Timestamp,
	totalLimit int,
) ([]*nostr.Event, error) {
	limitPerKey := PerQueryLimitInBatch(totalLimit, len(pubkeys))
	events := make([]*nostr.Event, 0, len(pubkeys)*limitPerKey)
	eventMap := make(map[string]struct{})

	wg := sync.WaitGroup{}
	wg.Add(len(pubkeys))

	for _, pubkey := range pubkeys {
		var latestTimestamp nostr.Timestamp
		var oldestTimestamp nostr.Timestamp
		var pkEvents []*nostr.Event
		oldestKey := makePubkeyStreamKey(pubkeyStreamOldestPrefix, pubkey)
		latestKey := makePubkeyStreamKey(pubkeyStreamLatestPrefix, pubkey)

		filter := nostr.Filter{Authors: []string{pubkey}, Kinds: kinds}
		filter.Until = &until
		if data, _ := sys.KVStore.Get(oldestKey); data != nil {
			oldestTimestamp = decodeTimestamp(data)
		}
		if data, _ := sys.KVStore.Get(latestKey); data != nil {
			latestTimestamp = decodeTimestamp(data)
		}

		if oldestTimestamp != 0 && oldestTimestamp < until && latestTimestamp != 0 && latestTimestamp > until {
			// Can query local store
			filter.Since = &oldestTimestamp
			res, err := sys.StoreRelay.QuerySync(ctx, filter)
			if err != nil {
				return nil, fmt.Errorf("query failure at '%s': %w", pubkey, err)
			}

			for _, e := range res {
				pkEvents = append(pkEvents, e)
				eventMap[e.ID] = struct{}{}
			}

			if len(pkEvents) >= limitPerKey {
				// we got enough from the local store
				wg.Done()
				continue
			} else {
				// need to look further back
				filter.Until = filter.Since
				filter.Since = nil
			}
		}

		relays := sys.FetchOutboxRelays(ctx, pubkey, 2)
		if len(relays) == 0 {
			wg.Done()
			continue
		}

		go func() {
			sub := sys.Pool.SubManyEose(ctx, relays, nostr.Filters{filter})

			for ie := range sub {
				if len(pkEvents) >= limitPerKey {
					break
				}
				if _, exists := eventMap[ie.ID]; !exists {
					sys.StoreRelay.Publish(ctx, *ie.Event)
					eventMap[ie.ID] = struct{}{}
					pkEvents = append(pkEvents, ie.Event)
				}
				if oldestTimestamp == 0 || ie.Event.CreatedAt < oldestTimestamp {
					oldestTimestamp = ie.Event.CreatedAt
				}
				if latestTimestamp == 0 || ie.Event.CreatedAt > latestTimestamp {
					latestTimestamp = ie.Event.CreatedAt
				}
			}

			events = append(events, pkEvents...)
			if oldestTimestamp != 0 {
				sys.KVStore.Set(oldestKey, encodeTimestamp(oldestTimestamp))
			}
			if latestTimestamp != 0 {
				sys.KVStore.Set(latestKey, encodeTimestamp(latestTimestamp))
			}
			wg.Done()
		}()
	}

	wg.Wait()
	slices.SortFunc(events, nostr.CompareEventPtrReverse)

	return events, nil
}
