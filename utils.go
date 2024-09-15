package nostr

import (
	"cmp"
	"encoding/hex"
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
	return true
}

func IsValid32ByteHex(thing string) bool {
	if strings.ToLower(thing) != thing {
		return false
	}
	if len(thing) != 64 {
		return false
	}
	_, err := hex.DecodeString(thing)
	return err == nil
}

func CompareEvent(a, b Event) int {
	if a.CreatedAt == b.CreatedAt {
		return strings.Compare(a.ID, b.ID)
	}
	return cmp.Compare(a.CreatedAt, b.CreatedAt)
}
