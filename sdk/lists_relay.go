package sdk

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
)

type Relay struct {
	URL    string
	Inbox  bool
	Outbox bool
}

func (r Relay) Value() string { return r.URL }

type RelayURL string

func (r RelayURL) Value() string { return string(r) }

func (sys *System) FetchRelayList(ctx context.Context, pubkey string) GenericList[Relay] {
	ml, _ := fetchGenericList(sys, ctx, pubkey, 10002, kind_10002, parseRelayFromKind10002, sys.RelayListCache, false)
	return ml
}

func (sys *System) FetchBlockedRelayList(ctx context.Context, pubkey string) GenericList[RelayURL] {
	ml, _ := fetchGenericList(sys, ctx, pubkey, 10006, kind_10006, parseRelayURL, sys.BlockedRelayListCache, false)
	return ml
}

func (sys *System) FetchSearchRelayList(ctx context.Context, pubkey string) GenericList[RelayURL] {
	ml, _ := fetchGenericList(sys, ctx, pubkey, 10007, kind_10007, parseRelayURL, sys.SearchRelayListCache, false)
	return ml
}

func (sys *System) FetchRelaySets(ctx context.Context, pubkey string) GenericSets[RelayURL] {
	ml, _ := fetchGenericSets(sys, ctx, pubkey, 30002, kind_30002, parseRelayURL, sys.RelaySetsCache, false)
	return ml
}

func parseRelayFromKind10002(tag nostr.Tag) (rl Relay, ok bool) {
	if u := tag.Value(); u != "" && tag[0] == "r" {
		if !nostr.IsValidRelayURL(u) {
			return rl, false
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

		return relay, true
	}

	return rl, false
}

func parseRelayURL(tag nostr.Tag) (rl RelayURL, ok bool) {
	if u := tag.Value(); u != "" && tag[0] == "relay" {
		if !nostr.IsValidRelayURL(u) {
			return rl, false
		}
		u := nostr.NormalizeURL(u)
		return RelayURL(u), true
	}

	return rl, false
}
