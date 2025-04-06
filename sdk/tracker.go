package sdk

import (
	"net/url"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip27"
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

// TrackEventHintsAndRelays is meant to be as an argument to WithEventMiddleware() when you're interested
// in tracking relays associated to event ids as well as feeding hints to the HintsDB.
func (sys *System) TrackEventHintsAndRelays(ie nostr.RelayEvent) {
	if IsVirtualRelay(ie.Relay.URL) {
		return
	}
	if ie.Kind < 30000 && ie.Kind >= 20000 {
		return
	}

	if ie.Kind != 0 && ie.Kind != 10002 {
		sys.trackEventRelay(ie.ID, ie.Relay.URL, false)
	}

	sys.trackEventHints(ie)
}

// TrackEventHints is meant to be used standalone as an argument to WithEventMiddleware() when you're not interested
// in tracking relays associated to event ids.
func (sys *System) TrackEventHints(ie nostr.RelayEvent) {
	if IsVirtualRelay(ie.Relay.URL) {
		return
	}
	if ie.Kind < 30000 && ie.Kind >= 20000 {
		return
	}

	sys.trackEventHints(ie)
}

func (sys *System) trackEventHints(ie nostr.RelayEvent) {
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
				sys.Hints.Save(ie.PubKey, nostr.NormalizeURL(tag[1]), hints.LastInRelayList, ie.CreatedAt)
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
				sys.Hints.Save(tag[1], nostr.NormalizeURL(tag[2]), hints.LastInHint, ie.CreatedAt)
			}
		}
	default:
		// everything else we track by relays and also check for hints
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
				sys.Hints.Save(tag[1], nostr.NormalizeURL(tag[2]), hints.LastInHint, ie.CreatedAt)
			}
		}

		for ref := range nip27.Parse(ie.Event.Content) {
			switch pointer := ref.Pointer.(type) {
			case nostr.ProfilePointer:
				for _, relay := range pointer.Relays {
					if IsVirtualRelay(relay) {
						continue
					}
					if p, err := url.Parse(relay); err != nil || (p.Scheme != "wss" && p.Scheme != "ws") {
						continue
					}
					sys.Hints.Save(pointer.PublicKey, nostr.NormalizeURL(relay), hints.LastInHint, ie.CreatedAt)
				}
			case nostr.EventPointer:
				for _, relay := range pointer.Relays {
					if IsVirtualRelay(relay) {
						continue
					}
					if p, err := url.Parse(relay); err != nil || (p.Scheme != "wss" && p.Scheme != "ws") {
						continue
					}
					sys.Hints.Save(pointer.Author, nostr.NormalizeURL(relay), hints.LastInHint, ie.CreatedAt)
				}
			case nostr.EntityPointer:
				for _, relay := range pointer.Relays {
					if IsVirtualRelay(relay) {
						continue
					}
					if p, err := url.Parse(relay); err != nil || (p.Scheme != "wss" && p.Scheme != "ws") {
						continue
					}
					sys.Hints.Save(pointer.PublicKey, nostr.NormalizeURL(relay), hints.LastInHint, ie.CreatedAt)
				}
			}
		}
	}
}

// TrackEventRelaysD is a companion to TrackEventRelays meant to be used with WithDuplicateMiddleware()
func (sys *System) TrackEventRelaysD(relay, id string) {
	if IsVirtualRelay(relay) {
		return
	}
	sys.trackEventRelay(id, relay, true /* we pass this flag so we'll skip creating entries for events that didn't pass the checks on the function above -- i.e. ephemeral events */)
}
