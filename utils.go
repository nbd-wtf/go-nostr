package nostr

import (
	"net/url"
	"strings"
)

func IsValidRelayURL(u string) bool {
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}
	if parsed.Scheme != "wss" && parsed.Scheme != "ws" {
		return false
	}
	if len(strings.Split(parsed.Host, ".")) < 2 {
		return false
	}
	return true
}
