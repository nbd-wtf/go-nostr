package sdk

import (
	"encoding/hex"
	"net/url"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/sdk/hints"
)

func (sys *System) TrackQueryAttempts(relay string, author string, kind int) {
	if IsVirtualRelay(relay) {
		return
	}
	if kind < 30000 && kind >= 20000 {
		return
	}
	if kind == 0 || kind == 10002 || kind == 3 {
		return
	}
	sys.Hints.Save(author, relay, hints.LastFetchAttempt, nostr.Now())
}

func (sys *System) TrackEventHints(ie nostr.RelayEvent) {
	if IsVirtualRelay(ie.Relay.URL) {
		return
	}
	if ie.Kind < 30000 && ie.Kind >= 20000 {
		return
	}

	switch ie.Kind {
	case nostr.KindProfileMetadata:
		// this could be anywhere so it doesn't count
		return
	case nostr.KindRelayListMetadata:
		// this is special, we only use it to track relay-list hints
		if len(ie.Tags) > 12 {
			// too many relays in the list means this person is not using this correctly so we better ignore them
			return
		}
		for _, tag := range ie.Tags {
			if len(tag) < 2 || tag[0] != "r" {
				continue
			}
			if len(tag) == 2 || (tag[2] == "" || tag[2] == "write") {
				sys.Hints.Save(ie.PubKey, tag[1], hints.LastInRelayList, ie.CreatedAt)
			}
		}
	case nostr.KindFollowList:
		// this is special, we only use it to check if there are hints for the contacts
		for _, tag := range ie.Tags {
			if len(tag) < 3 {
				continue
			}
			if IsVirtualRelay(tag[2]) {
				continue
			}
			if p, err := url.Parse(tag[2]); err != nil || (p.Scheme != "wss" && p.Scheme != "ws") {
				continue
			}
			if tag[0] == "p" && nostr.IsValidPublicKey(tag[1]) {
				sys.Hints.Save(tag[1], tag[2], hints.LastInTag, ie.CreatedAt)
			}
		}
	default:
		// everything else may have hints
		sys.Hints.Save(ie.PubKey, ie.Relay.URL, hints.MostRecentEventFetched, ie.CreatedAt)

		for _, tag := range ie.Tags {
			if len(tag) < 3 {
				continue
			}
			if IsVirtualRelay(tag[2]) {
				continue
			}
			if p, err := url.Parse(tag[2]); err != nil || (p.Scheme != "wss" && p.Scheme != "ws") {
				continue
			}
			if tag[0] == "p" && nostr.IsValidPublicKey(tag[1]) {
				sys.Hints.Save(tag[1], tag[2], hints.LastInTag, ie.CreatedAt)
			}
		}

		for ref := range ParseReferences(*ie.Event) {
			if ref.Profile != nil {
				for _, relay := range ref.Profile.Relays {
					if IsVirtualRelay(relay) {
						continue
					}
					if p, err := url.Parse(relay); err != nil || (p.Scheme != "wss" && p.Scheme != "ws") {
						continue
					}
					if nostr.IsValidPublicKey(ref.Profile.PublicKey) {
						sys.Hints.Save(ref.Profile.PublicKey, relay, hints.LastInNprofile, ie.CreatedAt)
					}
				}
			} else if ref.Event != nil && nostr.IsValidPublicKey(ref.Event.Author) {
				for _, relay := range ref.Event.Relays {
					if IsVirtualRelay(relay) {
						continue
					}
					if p, err := url.Parse(relay); err != nil || (p.Scheme != "wss" && p.Scheme != "ws") {
						continue
					}
					sys.Hints.Save(ref.Event.Author, relay, hints.LastInNevent, ie.CreatedAt)
				}
			}
		}
	}
}

const eventRelayPrefix = byte('r')

func makeEventRelayKey(eventID []byte, relay string) []byte {
	// Format: 'r' + first 8 bytes of event ID + relay URL
	key := make([]byte, 1+8+len(relay))
	key[0] = eventRelayPrefix
	copy(key[1:], eventID[:8])
	copy(key[9:], relay)
	return key
}

func (sys *System) TrackEventRelays(ie nostr.RelayEvent) {
	// decode the event ID hex into bytes
	idBytes, err := hex.DecodeString(ie.ID)
	if err != nil || len(idBytes) < 8 {
		return
	}

	// store with prefix + eventid + relay format
	key := makeEventRelayKey(idBytes, ie.Relay.URL)
	sys.KVStore.Set(key, nil) // value is not needed since relay is in key
}

func (sys *System) TrackEventRelaysD(relay, id string) {
	// decode the event ID hex into bytes
	idBytes, err := hex.DecodeString(id)
	if err != nil || len(idBytes) < 8 {
		return
	}

	// store with prefix + eventid + relay format
	key := makeEventRelayKey(idBytes, relay)
	sys.KVStore.Set(key, nil) // value is not needed since relay is in key
}
