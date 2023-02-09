package nip42

import (
	"net/url"
	"strings"
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

// helper function for ValidateAuthEvent
func parseUrl(input string) (*url.URL, error) {
	return url.Parse(
		strings.ToLower(
			strings.TrimSuffix(input, "/"),
		),
	)
}

// ValidateAuthEvent checks whether event is a valid NIP-42 event for given challenge and relayURL.
// The result of the validation is encoded in the ok bool.
func ValidateAuthEvent(event *nostr.Event, challenge string, relayURL string) (pubkey string, ok bool) {
	if event == nil || event.Kind != 22242 {
		return "", false
	}

	if event.Tags.GetFirst([]string{"challenge", challenge}) == nil {
		return "", false
	}

	expected, err := parseUrl(relayURL)
	if err != nil {
		return "", false
	}

	found, err := parseUrl(event.Tags.GetFirst([]string{"relay", ""}).Value())
	if err != nil {
		return "", false
	}

	if expected.Scheme != found.Scheme ||
		expected.Host != found.Host ||
		expected.Path != found.Path {
		return "", false
	}

	now := time.Now()
	if event.CreatedAt.After(now.Add(10*time.Minute)) || event.CreatedAt.Before(now.Add(-10*time.Minute)) {
		return "", false
	}

	// save for last, as it is most expensive operation
	if ok, err := event.CheckSignature(); !ok || err != nil {
		return "", false
	}

	return event.PubKey, true
}
