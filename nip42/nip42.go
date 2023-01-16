package nip42

import (
	"net/url"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

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
