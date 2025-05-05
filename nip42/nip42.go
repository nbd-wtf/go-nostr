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
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindClientAuthentication,
		Tags: nostr.Tags{
			nostr.Tag{"relay", relayURL},
			nostr.Tag{"challenge", challenge},
		},
		Content: "",
	}
}

// helper function for ValidateAuthEvent.
func parseURL(input string) (*url.URL, error) {
	return url.Parse(
		strings.ToLower(
			strings.TrimSuffix(input, "/"),
		),
	)
}

// ValidateAuthEvent checks whether event is a valid NIP-42 event for given challenge and relayURL.
// The result of the validation is encoded in the ok bool.
func ValidateAuthEvent(event *nostr.Event, challenge string, relayURL string) (pubkey string, ok bool) {
	if event.Kind != nostr.KindClientAuthentication {
		return "", false
	}

	if event.Tags.FindWithValue("challenge", challenge) == nil {
		return "", false
	}

	expected, err := parseURL(relayURL)
	if err != nil {
		return "", false
	}

	tag := event.Tags.Find("relay")
	if tag == nil {
		return "", false
	}

	found, err := parseURL(tag[1])
	if err != nil {
		return "", false
	}

	if expected.Scheme != found.Scheme ||
		expected.Host != found.Host ||
		expected.Path != found.Path {
		return "", false
	}

	now := time.Now()
	if event.CreatedAt.Time().After(now.Add(10*time.Minute)) || event.CreatedAt.Time().Before(now.Add(-10*time.Minute)) {
		return "", false
	}

	// save for last, as it is most expensive operation
	// no need to check returned error, since ok == true implies err == nil.
	if ok, _ := event.CheckSignature(); !ok {
		return "", false
	}

	return event.PubKey, true
}
