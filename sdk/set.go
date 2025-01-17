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
		sets := parseSetsFromEvents(events, parseTag)
		v.Events = events
		v.Sets = sets
		cache.SetWithTTL(pubkey, v, time.Hour*6)
		valueWasJustCached[lockIdx] = true
		return v, true
	}

	thunk := sys.addressableLoaders[addressableIndex].Load(ctx, pubkey)
	events, err := thunk()
	if err == nil {
		sets := parseSetsFromEvents(events, parseTag)
		v.Sets = sets
		for _, evt := range events {
			sys.StoreRelay.Publish(ctx, *evt)
		}
	}
	cache.SetWithTTL(pubkey, v, time.Hour*6)
	valueWasJustCached[lockIdx] = true

	return v, false
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
