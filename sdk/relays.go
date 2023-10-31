package sdk

import (
	"context"
	"encoding/json"

	"github.com/nbd-wtf/go-nostr"
)

type Relay struct {
	URL    string
	Inbox  bool
	Outbox bool
}

func FetchRelaysForPubkey(ctx context.Context, pool *nostr.SimplePool, pubkey string, relays ...string) []Relay {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := pool.SubManyEose(ctx, relays, nostr.Filters{
		{
			Kinds: []int{
				nostr.KindRelayListMetadata,
				nostr.KindContactList,
			},
			Authors: []string{pubkey},
			Limit:   2,
		},
	})

	result := make([]Relay, 0, 20)
	i := 0
	for ie := range ch {
		switch ie.Event.Kind {
		case nostr.KindRelayListMetadata:
			result = append(result, ParseRelaysFromKind10002(ie.Event)...)
		case nostr.KindContactList:
			result = append(result, ParseRelaysFromKind3(ie.Event)...)
		}

		i++
		if i >= 2 {
			break
		}
	}

	return result
}

func ParseRelaysFromKind10002(evt *nostr.Event) []Relay {
	result := make([]Relay, 0, len(evt.Tags))
	for _, tag := range evt.Tags {
		if u := tag.Value(); u != "" && tag[0] == "r" {
			if !nostr.IsValidRelayURL(u) {
				continue
			}
			u := nostr.NormalizeURL(u)

			relay := Relay{
				URL: u,
			}

			if len(tag) == 2 {
				relay.Inbox = true
				relay.Outbox = true
			} else if tag[2] == "write" {
				relay.Outbox = true
			} else if tag[2] == "read" {
				relay.Inbox = true
			}

			result = append(result, relay)
		}
	}

	return result
}

func ParseRelaysFromKind3(evt *nostr.Event) []Relay {
	type Item struct {
		Read  bool `json:"read"`
		Write bool `json:"write"`
	}

	items := make(map[string]Item, 20)
	json.Unmarshal([]byte(evt.Content), &items)

	results := make([]Relay, len(items))
	i := 0
	for u, item := range items {
		if !nostr.IsValidRelayURL(u) {
			continue
		}
		u := nostr.NormalizeURL(u)

		relay := Relay{
			URL: u,
		}

		if item.Read {
			relay.Inbox = true
		}
		if item.Write {
			relay.Outbox = true
		}

		results = append(results, relay)
		i++
	}

	return results
}
