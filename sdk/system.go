package sdk

import (
	"context"
	"math/rand/v2"

	"github.com/fiatjaf/eventstore"
	"github.com/fiatjaf/eventstore/nullstore"
	"github.com/graph-gophers/dataloader/v7"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/sdk/cache"
	cache_memory "github.com/nbd-wtf/go-nostr/sdk/cache/memory"
	"github.com/nbd-wtf/go-nostr/sdk/hints"
	"github.com/nbd-wtf/go-nostr/sdk/hints/memoryh"
)

type System struct {
	MetadataCache         cache.Cache32[ProfileMetadata]
	RelayListCache        cache.Cache32[GenericList[Relay]]
	FollowListCache       cache.Cache32[GenericList[ProfileRef]]
	MuteListCache         cache.Cache32[GenericList[ProfileRef]]
	BookmarkListCache     cache.Cache32[GenericList[EventRef]]
	PinListCache          cache.Cache32[GenericList[EventRef]]
	BlockedRelayListCache cache.Cache32[GenericList[RelayURL]]
	SearchRelayListCache  cache.Cache32[GenericList[RelayURL]]
	Hints                 hints.HintsDB
	Pool                  *nostr.SimplePool
	RelayListRelays       *RelayStream
	FollowListRelays      *RelayStream
	MetadataRelays        *RelayStream
	FallbackRelays        *RelayStream
	JustIDRelays          *RelayStream
	UserSearchRelays      *RelayStream
	NoteSearchRelays      *RelayStream
	Store                 eventstore.Store

	StoreRelay nostr.RelayStore

	replaceableLoaders   []*dataloader.Loader[string, *nostr.Event]
	outboxShortTermCache cache.Cache32[[]string]
}

type SystemModifier func(sys *System)

type RelayStream struct {
	URLs   []string
	serial int
}

func NewRelayStream(urls ...string) *RelayStream {
	return &RelayStream{URLs: urls, serial: rand.Int()}
}

func (rs *RelayStream) Next() string {
	rs.serial++
	return rs.URLs[rs.serial%len(rs.URLs)]
}

func NewSystem(mods ...SystemModifier) *System {
	sys := &System{
		MetadataCache:         cache_memory.New32[ProfileMetadata](8000),
		RelayListCache:        cache_memory.New32[GenericList[Relay]](8000),
		FollowListCache:       cache_memory.New32[GenericList[ProfileRef]](1000),
		MuteListCache:         cache_memory.New32[GenericList[ProfileRef]](1000),
		BookmarkListCache:     cache_memory.New32[GenericList[EventRef]](1000),
		PinListCache:          cache_memory.New32[GenericList[EventRef]](1000),
		BlockedRelayListCache: cache_memory.New32[GenericList[RelayURL]](1000),
		SearchRelayListCache:  cache_memory.New32[GenericList[RelayURL]](1000),
		RelayListRelays:       NewRelayStream("wss://purplepag.es", "wss://user.kindpag.es", "wss://relay.nos.social"),
		FollowListRelays:      NewRelayStream("wss://purplepag.es", "wss://user.kindpag.es", "wss://relay.nos.social"),
		MetadataRelays:        NewRelayStream("wss://purplepag.es", "wss://user.kindpag.es", "wss://relay.nos.social"),
		FallbackRelays: NewRelayStream(
			"wss://relay.damus.io",
			"wss://nostr.mom",
			"wss://nos.lol",
			"wss://mostr.pub",
			"wss://relay.nostr.band",
		),
		JustIDRelays: NewRelayStream(
			"wss://cache2.primal.net/v1",
			"wss://relay.noswhere.com",
			"wss://relay.nostr.band",
		),
		UserSearchRelays: NewRelayStream(
			"wss://search.nos.today",
			"wss://nostr.wine",
			"wss://relay.nostr.band",
		),
		NoteSearchRelays: NewRelayStream(
			"wss://nostr.wine",
			"wss://relay.nostr.band",
			"wss://search.nos.today",
		),
		Hints: memoryh.NewHintDB(),

		outboxShortTermCache: cache_memory.New32[[]string](1000),
	}

	sys.Pool = nostr.NewSimplePool(context.Background(),
		nostr.WithQueryMiddleware(sys.TrackQueryAttempts),
		nostr.WithEventMiddleware(sys.TrackEventHints),
		nostr.WithPenaltyBox(),
	)

	for _, mod := range mods {
		mod(sys)
	}

	if sys.Store == nil {
		sys.Store = &nullstore.NullStore{}
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
		sys.RelayListRelays.URLs = list
	}
}

func WithMetadataRelays(list []string) SystemModifier {
	return func(sys *System) {
		sys.MetadataRelays.URLs = list
	}
}

func WithFollowListRelays(list []string) SystemModifier {
	return func(sys *System) {
		sys.FollowListRelays.URLs = list
	}
}

func WithFallbackRelays(list []string) SystemModifier {
	return func(sys *System) {
		sys.FallbackRelays.URLs = list
	}
}

func WithJustIDRelays(list []string) SystemModifier {
	return func(sys *System) {
		sys.JustIDRelays.URLs = list
	}
}

func WithUserSearchRelays(list []string) SystemModifier {
	return func(sys *System) {
		sys.UserSearchRelays.URLs = list
	}
}

func WithNoteSearchRelays(list []string) SystemModifier {
	return func(sys *System) {
		sys.NoteSearchRelays.URLs = list
	}
}

func WithStore(store eventstore.Store) SystemModifier {
	return func(sys *System) {
		sys.Store = store
	}
}

func WithRelayListCache(cache cache.Cache32[GenericList[Relay]]) SystemModifier {
	return func(sys *System) {
		sys.RelayListCache = cache
	}
}

func WithFollowListCache(cache cache.Cache32[GenericList[ProfileRef]]) SystemModifier {
	return func(sys *System) {
		sys.FollowListCache = cache
	}
}

func WithMetadataCache(cache cache.Cache32[ProfileMetadata]) SystemModifier {
	return func(sys *System) {
		sys.MetadataCache = cache
	}
}
