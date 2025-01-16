package sdk

import (
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

func (sys *System) TrackEventHintsAndRelays(ie nostr.RelayEvent) {
	if IsVirtualRelay(ie.Relay.URL) {
		return
	}
	if ie.Kind < 30000 && ie.Kind >= 20000 {
		return
	}

	if ie.Kind != 0 && ie.Kind != 10002 {
		sys.trackEventRelayCommon(ie.ID, ie.Relay.URL, false)
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
				sys.Hints.Save(tag[1], nostr.NormalizeURL(tag[2]), hints.LastInTag, ie.CreatedAt)
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
				sys.Hints.Save(tag[1], nostr.NormalizeURL(tag[2]), hints.LastInTag, ie.CreatedAt)
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
						sys.Hints.Save(ref.Profile.PublicKey, nostr.NormalizeURL(relay), hints.LastInNprofile, ie.CreatedAt)
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
					sys.Hints.Save(ref.Event.Author, nostr.NormalizeURL(relay), hints.LastInNevent, ie.CreatedAt)
				}
			}
		}
	}
}

func (sys *System) TrackEventRelaysD(relay, id string) {
	if IsVirtualRelay(relay) {
		return
	}
	sys.trackEventRelayCommon(id, relay, true /* we pass this flag so we'll skip creating entries for events that didn't pass the checks on the function above -- i.e. ephemeral events */)
}
