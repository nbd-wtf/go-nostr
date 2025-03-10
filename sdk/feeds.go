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
			sub := sys.Pool.SubscribeMany(ctx, relays, filter, nostr.WithLabel("livefeed"))
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

	wg := sync.WaitGroup{}
	wg.Add(len(pubkeys))

	for _, pubkey := range pubkeys {
		oldestKey := makePubkeyStreamKey(pubkeyStreamOldestPrefix, pubkey)
		var oldestTimestamp nostr.Timestamp

		if data, _ := sys.KVStore.Get(oldestKey); data != nil {
			oldestTimestamp = decodeTimestamp(data)
			if oldestTimestamp == 0 {
				oldestTimestamp = nostr.Now()
			}
		}

		filter := nostr.Filter{Authors: []string{pubkey}, Kinds: kinds}

		if until > oldestTimestamp {
			// we can use our local database
			filter.Until = &until
			res, err := sys.StoreRelay.QuerySync(ctx, filter)
			if err != nil {
				return nil, fmt.Errorf("query failure at '%s': %w", pubkey, err)
			}

			if len(res) >= limitPerKey {
				// we got enough from the local store
				events = append(events, res...)
				wg.Done()
				continue
			}
		}

		// if we didn't get enough events from local database
		// OR if we are requesting for very old stuff
		// then we will query relays -- always with Until set to our oldestTimestamp+1
		// (so we don't get events we already have)
		relays := sys.FetchOutboxRelays(ctx, pubkey, 2)
		if len(relays) == 0 {
			wg.Done()
			continue
		}
		fUntil := oldestTimestamp + 1
		filter.Until = &fUntil
		filter.Since = nil
		for ie := range sys.Pool.FetchMany(ctx, relays, filter, nostr.WithLabel("feedpage")) {
			sys.StoreRelay.Publish(ctx, *ie.Event)

			// we shouldn't need this check here, but against rogue relays we'll do it
			if ie.Event.CreatedAt < oldestTimestamp {
				oldestTimestamp = ie.Event.CreatedAt
			}

			// we should check this because we might be just catching up to the point where the
			// offset that was requested.
			// so we don't add these events to our results, just to our local store (above)
			if ie.Event.CreatedAt < until {
				events = append(events, ie.Event)
			}
		}
		wg.Done()
		sys.KVStore.Set(oldestKey, encodeTimestamp(oldestTimestamp))
	}

	wg.Wait()
	slices.SortFunc(events, nostr.CompareEventPtrReverse)

	return events, nil
}
