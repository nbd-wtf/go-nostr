package sdk

import (
	"context"
	"slices"
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

func fetchGenericList[I TagItemWithValue](
	sys *System,
	ctx context.Context,
	pubkey string,
	kind int,
	parseTag func(nostr.Tag) (I, bool),
	cache cache.Cache32[GenericList[I]],
	skipFetch bool,
) (fl GenericList[I], fromInternal bool) {
	if cache != nil {
		if v, ok := cache.Get(pubkey); ok {
			return v, true
		}
	}

	events, _ := sys.StoreRelay.QuerySync(ctx, nostr.Filter{Kinds: []int{kind}, Authors: []string{pubkey}})
	if len(events) != 0 {
		items := parseItemsFromEventTags(events[0], parseTag)
		v := GenericList[I]{
			PubKey: pubkey,
			Event:  events[0],
			Items:  items,
		}
		cache.SetWithTTL(pubkey, v, time.Hour*6)
		return v, true
	}

	v := GenericList[I]{PubKey: pubkey}
	if !skipFetch {
		thunk := sys.replaceableLoaders[kind].Load(ctx, pubkey)
		evt, err := thunk()
		if err == nil {
			items := parseItemsFromEventTags(evt, parseTag)
			v.Items = items
			if cache != nil {
				cache.SetWithTTL(pubkey, v, time.Hour*6)
			}
			sys.StoreRelay.Publish(ctx, *evt)
		}
	}

	return v, false
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
