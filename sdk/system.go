package sdk

import (
	"context"

	"github.com/fiatjaf/eventstore"
	"github.com/fiatjaf/eventstore/slicestore"
	"github.com/graph-gophers/dataloader/v7"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/sdk/cache"
	cache_memory "github.com/nbd-wtf/go-nostr/sdk/cache/memory"
	"github.com/nbd-wtf/go-nostr/sdk/hints"
	memory_hints "github.com/nbd-wtf/go-nostr/sdk/hints/memory"
)

type System struct {
	RelayListCache   cache.Cache32[RelayList]
	FollowListCache  cache.Cache32[FollowList]
	MetadataCache    cache.Cache32[ProfileMetadata]
	Hints            hints.HintsDB
	Pool             *nostr.SimplePool
	RelayListRelays  []string
	FollowListRelays []string
	MetadataRelays   []string
	FallbackRelays   []string
	UserSearchRelays []string
	NoteSearchRelays []string
	Store            eventstore.Store

	StoreRelay nostr.RelayStore

	replaceableLoaders   map[int]*dataloader.Loader[string, *nostr.Event]
	outboxShortTermCache cache.Cache32[[]string]
}

type SystemModifier func(sys *System)

func NewSystem(mods ...SystemModifier) *System {
	sys := &System{
		RelayListCache:   cache_memory.New32[RelayList](1000),
		FollowListCache:  cache_memory.New32[FollowList](1000),
		MetadataCache:    cache_memory.New32[ProfileMetadata](1000),
		RelayListRelays:  []string{"wss://purplepag.es", "wss://user.kindpag.es", "wss://relay.nos.social"},
		FollowListRelays: []string{"wss://purplepag.es", "wss://user.kindpag.es", "wss://relay.nos.social"},
		MetadataRelays:   []string{"wss://purplepag.es", "wss://user.kindpag.es", "wss://relay.nos.social"},
		FallbackRelays: []string{
			"wss://relay.primal.net",
			"wss://relay.damus.io",
			"wss://nostr.wine",
			"wss://nostr.mom",
			"wss://offchain.pub",
			"wss://nos.lol",
			"wss://mostr.pub",
			"wss://relay.nostr.band",
			"wss://nostr21.com",
		},
		UserSearchRelays: []string{
			"wss://nostr.wine",
			"wss://relay.nostr.band",
			"wss://relay.noswhere.com",
		},
		NoteSearchRelays: []string{
			"wss://nostr.wine",
			"wss://relay.nostr.band",
			"wss://relay.noswhere.com",
		},
		Hints: memory_hints.NewHintDB(),

		outboxShortTermCache: cache_memory.New32[[]string](1000),
	}

	sys.Pool = nostr.NewSimplePool(context.Background(),
		nostr.WithEventMiddleware(sys.trackEventHints),
		nostr.WithPenaltyBox(),
	)

	for _, mod := range mods {
		mod(sys)
	}

	if sys.Store == nil {
		sys.Store = &slicestore.SliceStore{}
		sys.Store.Init()
	}
	sys.StoreRelay = eventstore.RelayWrapper{Store: sys.Store}

	sys.initializeDataloaders()

	return sys
}

func (sys *System) Close() {}

func WithHintsDB(hdb hints.HintsDB) SystemModifier {
	return func(sys *System) {
		sys.Hints = hdb
	}
}

func WithRelayListRelays(list []string) SystemModifier {
	return func(sys *System) {
		sys.RelayListRelays = list
	}
}

func WithMetadataRelays(list []string) SystemModifier {
	return func(sys *System) {
		sys.MetadataRelays = list
	}
}

func WithFollowListRelays(list []string) SystemModifier {
	return func(sys *System) {
		sys.FollowListRelays = list
	}
}

func WithFallbackRelays(list []string) SystemModifier {
	return func(sys *System) {
		sys.FallbackRelays = list
	}
}

func WithUserSearchRelays(list []string) SystemModifier {
	return func(sys *System) {
		sys.UserSearchRelays = list
	}
}

func WithNoteSearchRelays(list []string) SystemModifier {
	return func(sys *System) {
		sys.NoteSearchRelays = list
	}
}

func WithStore(store eventstore.Store) SystemModifier {
	return func(sys *System) {
		sys.Store = store
	}
}

func WithRelayListCache(cache cache.Cache32[RelayList]) SystemModifier {
	return func(sys *System) {
		sys.RelayListCache = cache
	}
}

func WithFollowListCache(cache cache.Cache32[FollowList]) SystemModifier {
	return func(sys *System) {
		sys.FollowListCache = cache
	}
}

func WithMetadataCache(cache cache.Cache32[ProfileMetadata]) SystemModifier {
	return func(sys *System) {
		sys.MetadataCache = cache
	}
}
