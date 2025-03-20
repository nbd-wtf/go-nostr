package sdk

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/sdk/cache"
)

// this is similar to list.go and inherits code from that.

type GenericSets[I TagItemWithValue] struct {
	PubKey string         `json:"-"`
	Events []*nostr.Event `json:"-"`

	Sets map[string][]I
}

func fetchGenericSets[I TagItemWithValue](
	sys *System,
	ctx context.Context,
	pubkey string,
	actualKind int,
	addressableIndex addressableIndex,
	parseTag func(nostr.Tag) (I, bool),
	cache cache.Cache32[GenericSets[I]],
) (fl GenericSets[I], fromInternal bool) {
	n, _ := strconv.ParseUint(pubkey[14:16], 16, 8)
	lockIdx := (n + uint64(actualKind)) % 60
	genericListMutexes[lockIdx].Lock()

	if valueWasJustCached[lockIdx] {
		// this ensures the cache has had time to commit the values
		// so we don't repeat a fetch immediately after the other
		valueWasJustCached[lockIdx] = false
		time.Sleep(time.Millisecond * 10)
	}

	genericListMutexes[lockIdx].Unlock()

	if v, ok := cache.Get(pubkey); ok {
		return v, true
	}

	v := GenericSets[I]{PubKey: pubkey}

	events, _ := sys.StoreRelay.QuerySync(ctx, nostr.Filter{Kinds: []int{actualKind}, Authors: []string{pubkey}})
	if len(events) != 0 {
		// ok, we found something locally
		sets := parseSetsFromEvents(events, parseTag)
		v.Events = events
		v.Sets = sets

		// but if we haven't tried fetching from the network recently we should do it
		lastFetchKey := makeLastFetchKey(actualKind, pubkey)
		lastFetchData, _ := sys.KVStore.Get(lastFetchKey)
		if lastFetchData == nil || nostr.Now()-decodeTimestamp(lastFetchData) > getLocalStoreRefreshDaysForKind(actualKind)*24*60*60 {
			newV := tryFetchSetsFromNetwork(ctx, sys, pubkey, addressableIndex, parseTag)

			// unlike for lists, when fetching sets we will blindly trust whatever we get from the network
			v = *newV

			// even if we didn't find anything register this because we tried
			// (and we still have the previous event in our local store)
			sys.KVStore.Set(lastFetchKey, encodeTimestamp(nostr.Now()))
		}

		// and finally save this to cache
		cache.SetWithTTL(pubkey, v, time.Hour*6)
		valueWasJustCached[lockIdx] = true

		return v, true
	}

	if newV := tryFetchSetsFromNetwork(ctx, sys, pubkey, addressableIndex, parseTag); newV != nil {
		v = *newV

		// we'll only save this if we got something which means we found at least one event
		lastFetchKey := makeLastFetchKey(actualKind, pubkey)
		sys.KVStore.Set(lastFetchKey, encodeTimestamp(nostr.Now()))
	}

	// save cache even if we didn't get anything
	cache.SetWithTTL(pubkey, v, time.Hour*6)
	valueWasJustCached[lockIdx] = true

	return v, false
}

func tryFetchSetsFromNetwork[I TagItemWithValue](
	ctx context.Context,
	sys *System,
	pubkey string,
	addressableIndex addressableIndex,
	parseTag func(nostr.Tag) (I, bool),
) *GenericSets[I] {
	events, err := sys.addressableLoaders[addressableIndex].Load(ctx, pubkey)
	if err != nil {
		return nil
	}

	v := &GenericSets[I]{
		PubKey: pubkey,
		Events: events,
		Sets:   parseSetsFromEvents(events, parseTag),
	}
	for _, evt := range events {
		sys.StoreRelay.Publish(ctx, *evt)
	}
	return v
}

func parseSetsFromEvents[I TagItemWithValue](
	events []*nostr.Event,
	parseTag func(nostr.Tag) (I, bool),
) map[string][]I {
	sets := make(map[string][]I, len(events))
	for _, evt := range events {
		items := make([]I, 0, len(evt.Tags))
		for _, tag := range evt.Tags {
			item, ok := parseTag(tag)
			if ok {
				// check if this already exists before adding
				if slices.IndexFunc(items, func(i I) bool { return i.Value() == item.Value() }) == -1 {
					items = append(items, item)
				}
			}
		}
		sets[evt.Tags.GetD()] = items
	}
	return sets
}
