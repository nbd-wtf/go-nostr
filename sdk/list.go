package sdk

import (
	"context"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/sdk/cache"
)

type GenericList[I TagItemWithValue] struct {
	PubKey string       `json:"-"` // must always be set otherwise things will break
	Event  *nostr.Event `json:"-"` // may be empty if a contact list event wasn't found

	Items []I
}

type TagItemWithValue interface {
	Value() string
}

var (
	genericListMutexes = [60]sync.Mutex{}
	valueWasJustCached = [60]bool{}
)

func fetchGenericList[I TagItemWithValue](
	sys *System,
	ctx context.Context,
	pubkey string,
	actualKind int,
	replaceableIndex replaceableIndex,
	parseTag func(nostr.Tag) (I, bool),
	cache cache.Cache32[GenericList[I]],
) (fl GenericList[I], fromInternal bool) {
	// we have 60 mutexes, so we can load up to 60 lists at the same time, but if we do the same exact
	// call that will do it only once, the subsequent ones will wait for a result to be cached
	// and then return it from cache -- 13 is an arbitrary index for the pubkey
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

	v := GenericList[I]{PubKey: pubkey}

	events, _ := sys.StoreRelay.QuerySync(ctx, nostr.Filter{Kinds: []int{actualKind}, Authors: []string{pubkey}})
	if len(events) != 0 {
		// ok, we found something locally
		items := parseItemsFromEventTags(events[0], parseTag)
		v.Event = events[0]
		v.Items = items

		// but if we haven't tried fetching from the network recently we should do it
		lastFetchKey := makeLastFetchKey(actualKind, pubkey)
		lastFetchData, _ := sys.KVStore.Get(lastFetchKey)
		if lastFetchData == nil || nostr.Now()-decodeTimestamp(lastFetchData) > getLocalStoreRefreshDaysForKind(actualKind)*24*60*60 {
			newV := tryFetchListFromNetwork(ctx, sys, pubkey, replaceableIndex, parseTag)
			if newV != nil && newV.Event.CreatedAt > v.Event.CreatedAt {
				v = *newV
			}

			// register this even if we didn't find anything because we tried
			// (and we still have the previous event in our local store)
			sys.KVStore.Set(lastFetchKey, encodeTimestamp(nostr.Now()))
		}

		// and finally save this to cache
		cache.SetWithTTL(pubkey, v, time.Hour*6)
		valueWasJustCached[lockIdx] = true

		return v, true
	}

	if newV := tryFetchListFromNetwork(ctx, sys, pubkey, replaceableIndex, parseTag); newV != nil {
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

func tryFetchListFromNetwork[I TagItemWithValue](
	ctx context.Context,
	sys *System,
	pubkey string,
	replaceableIndex replaceableIndex,
	parseTag func(nostr.Tag) (I, bool),
) *GenericList[I] {
	evt, err := sys.replaceableLoaders[replaceableIndex].Load(ctx, pubkey)
	if err != nil {
		return nil
	}

	v := &GenericList[I]{
		PubKey: pubkey,
		Event:  evt,
		Items:  parseItemsFromEventTags(evt, parseTag),
	}
	sys.StoreRelay.Publish(ctx, *evt)

	return v
}

func parseItemsFromEventTags[I TagItemWithValue](
	evt *nostr.Event,
	parseTag func(nostr.Tag) (I, bool),
) []I {
	result := make([]I, 0, len(evt.Tags))
	for _, tag := range evt.Tags {
		item, ok := parseTag(tag)
		if ok {
			// check if this already exists before adding
			if slices.IndexFunc(result, func(i I) bool { return i.Value() == item.Value() }) == -1 {
				result = append(result, item)
			}
		}
	}
	return result
}

func getLocalStoreRefreshDaysForKind(kind int) nostr.Timestamp {
	switch kind {
	case 0:
		return 7
	case 3:
		return 1
	default:
		return 3
	}
}
