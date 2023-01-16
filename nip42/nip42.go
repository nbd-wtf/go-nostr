package nip42

import (
	"net/url"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

// CreateUnsignedAuthEvent creates an event which should be sent via an "AUTH" command.
// If the authentication succeeds, the user will be authenticated as pubkey.
func CreateUnsignedAuthEvent(challenge, pubkey, relayURL string) nostr.Event {
	return nostr.Event{
		PubKey:    pubkey,
		CreatedAt: time.Now(),
		Kind:      22242,
		Tags: nostr.Tags{
			nostr.Tag{"relay", relayURL},
			nostr.Tag{"challenge", challenge},
		},
		Content: "",
	}
}

// ValidateAuthEvent checks whether event is a valid NIP-42 event for given challenge and relayURL.
// The result of the validation is encoded in the ok bool.
func ValidateAuthEvent(event *nostr.Event, challenge string, relayURL string) (pubkey string, ok bool) {
	if ok, _ := event.CheckSignature(); ok == false {
		return "", false
	}
	if event.Kind != 22242 {
		return "", false
	}

	now := time.Now()
	if event.CreatedAt.After(now.Add(10*time.Minute)) || event.CreatedAt.Before(now.Add(-10*time.Minute)) {
		return "", false
	}

	if event.Tags.GetFirst([]string{"challenge", challenge}) == nil {
		return "", false
	}

	expected, err1 := url.Parse(relayURL)
	found, err2 := url.Parse(event.Tags.GetFirst([]string{"relay", ""}).Value())
	if err1 != nil || err2 != nil {
		return "", false
	} else {
		if expected.Scheme != found.Scheme ||
			expected.Host != found.Host ||
			expected.Path != found.Path {
			return "", false
		}
	}

	return event.PubKey, true
}
