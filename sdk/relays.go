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

func FetchRelaysForPubkey(ctx context.Context, pool *nostr.SimplePool, pubkey string, extraRelays ...string) []Relay {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	relays := append(extraRelays,
		"wss://nostr-pub.wellorder.net",
		"wss://relay.damus.io",
		"wss://nos.lol",
		"wss://nostr.mom",
		"wss://relay.nostr.bg",
	)

	ch := pool.SubManyEose(ctx, relays, nostr.Filters{
		{
			Kinds:   []int{10002, 3},
			Authors: []string{pubkey},
			Limit:   2,
		},
	})

	result := make([]Relay, 0, 20)
	i := 0
	for event := range ch {
		switch event.Kind {
		case 10002:
			result = append(result, ParseRelaysFromKind10002(event)...)
		case 3:
			result = append(result, ParseRelaysFromKind3(event)...)
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
