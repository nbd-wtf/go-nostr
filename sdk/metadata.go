package sdk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip05"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/go-nostr/sdk/hints"
)

type ProfileMetadata struct {
	PubKey string       `json:"-"` // must always be set otherwise things will break
	Event  *nostr.Event `json:"-"` // may be empty if a profile metadata event wasn't found

	// every one of these may be empty
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	About       string `json:"about,omitempty"`
	Website     string `json:"website,omitempty"`
	Picture     string `json:"picture,omitempty"`
	Banner      string `json:"banner,omitempty"`
	NIP05       string `json:"nip05,omitempty"`
	LUD16       string `json:"lud16,omitempty"`

	nip05Valid       bool
	nip05LastAttempt time.Time
}

func (p ProfileMetadata) Npub() string {
	v, _ := nip19.EncodePublicKey(p.PubKey)
	return v
}

func (p ProfileMetadata) NpubShort() string {
	npub := p.Npub()
	return npub[0:7] + "…" + npub[58:]
}

func (p ProfileMetadata) Nprofile(ctx context.Context, sys *System, nrelays int) string {
	v, _ := nip19.EncodeProfile(p.PubKey, sys.FetchOutboxRelays(ctx, p.PubKey, 2))
	return v
}

func (p ProfileMetadata) ShortName() string {
	if p.Name != "" {
		return p.Name
	}
	if p.DisplayName != "" {
		return p.DisplayName
	}
	return p.NpubShort()
}

func (p *ProfileMetadata) NIP05Valid(ctx context.Context) bool {
	if p.NIP05 == "" {
		return false
	}

	now := time.Now()
	if p.nip05LastAttempt.Before(now.AddDate(0, 0, -7)) {
		// must revalidate
		p.nip05LastAttempt = now
		pp, err := nip05.QueryIdentifier(ctx, p.NIP05)
		if err != nil {
			p.nip05Valid = false
		} else {
			p.nip05Valid = pp.PublicKey == p.PubKey
		}
	}
	return p.nip05Valid
}

// FetchProfileFromInput takes an nprofile, npub, nip05 or hex pubkey and returns a ProfileMetadata,
// updating the hintsDB in the process with any eventual relay hints
func (sys System) FetchProfileFromInput(ctx context.Context, nip19OrNip05Code string) (ProfileMetadata, error) {
	p := InputToProfile(ctx, nip19OrNip05Code)
	if p == nil {
		return ProfileMetadata{}, fmt.Errorf("couldn't decode profile reference")
	}

	for _, r := range p.Relays {
		if !IsVirtualRelay(r) {
			sys.Hints.Save(p.PublicKey, nostr.NormalizeURL(r), hints.LastInHint, nostr.Now())
		}
	}

	pm := sys.FetchProfileMetadata(ctx, p.PublicKey)
	return pm, nil
}

// FetchProfileMetadata fetches metadata for a given user from the local cache, or from the local store,
// or, failing these, from the target user's defined outbox relays -- then caches the result.
func (sys *System) FetchProfileMetadata(ctx context.Context, pubkey string) (pm ProfileMetadata) {
	if v, ok := sys.MetadataCache.Get(pubkey); ok {
		return v
	}

	res, _ := sys.StoreRelay.QuerySync(ctx, nostr.Filter{Kinds: []int{0}, Authors: []string{pubkey}})
	if len(res) != 0 {
		// ok, we found something locally
		pm, _ = ParseMetadata(res[0])
		pm.PubKey = pubkey
		pm.Event = res[0]

		// but if we haven't tried fetching from the network recently we should do it
		lastFetchKey := makeLastFetchKey(0, pubkey)
		lastFetchData, _ := sys.KVStore.Get(lastFetchKey)
		if nostr.Now()-decodeTimestamp(lastFetchData) > 7*24*60*60 {
			newM := sys.tryFetchMetadataFromNetwork(ctx, pubkey)
			if newM != nil && newM.Event.CreatedAt > pm.Event.CreatedAt {
				pm = *newM
			}

			// even if we didn't find anything register this because we tried
			// (and we still have the previous event in our local store)
			sys.KVStore.Set(lastFetchKey, encodeTimestamp(nostr.Now()))
		}

		// and finally save this to cache
		sys.MetadataCache.SetWithTTL(pubkey, pm, time.Hour*6)

		return pm
	}

	if newM := sys.tryFetchMetadataFromNetwork(ctx, pubkey); newM != nil {
		pm = *newM

		// we'll only save this if we got something which means we found at least one event
		lastFetchKey := makeLastFetchKey(0, pubkey)
		sys.KVStore.Set(lastFetchKey, encodeTimestamp(nostr.Now()))
	}

	// save cache even if we didn't get anything
	sys.MetadataCache.SetWithTTL(pubkey, pm, time.Hour*6)

	return pm
}

// FetchUserEvents fetches events from each users' outbox relays, grouping queries when possible.
func (sys *System) FetchUserEvents(ctx context.Context, filter nostr.Filter) (map[string][]*nostr.Event, error) {
	filters, err := sys.ExpandQueriesByAuthorAndRelays(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to expand queries: %w", err)
	}

	results := make(map[string][]*nostr.Event)
	wg := sync.WaitGroup{}
	wg.Add(len(filters))
	for relayURL, filter := range filters {
		go func(relayURL string, filter nostr.Filter) {
			defer wg.Done()
			filter.Limit = filter.Limit * len(filter.Authors) // hack
			for ie := range sys.Pool.SubManyEose(ctx, []string{relayURL}, nostr.Filters{filter}, nostr.WithLabel("userevts")) {
				results[ie.PubKey] = append(results[ie.PubKey], ie.Event)
			}
		}(relayURL, filter)
	}
	wg.Wait()

	return results, nil
}

func (sys *System) tryFetchMetadataFromNetwork(ctx context.Context, pubkey string) *ProfileMetadata {
	thunk0 := sys.replaceableLoaders[kind_0].Load(ctx, pubkey)
	evt, err := thunk0()
	if err != nil {
		return nil
	}

	pm, err := ParseMetadata(evt)
	if err != nil {
		return nil
	}

	pm.PubKey = pubkey
	pm.Event = evt
	sys.StoreRelay.Publish(ctx, *evt)
	sys.MetadataCache.SetWithTTL(pubkey, pm, time.Hour*6)
	return &pm
}

func ParseMetadata(event *nostr.Event) (meta ProfileMetadata, err error) {
	if event.Kind != 0 {
		err = fmt.Errorf("event %s is kind %d, not 0", event.ID, event.Kind)
	} else if er := json.Unmarshal([]byte(event.Content), &meta); er != nil {
		cont := event.Content
		if len(cont) > 100 {
			cont = cont[0:99]
		}
		err = fmt.Errorf("failed to parse metadata (%s) from event %s: %w", cont, event.ID, er)
	}

	meta.PubKey = event.PubKey
	meta.Event = event
	return meta, err
}
