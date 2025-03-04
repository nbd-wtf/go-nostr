package nostr

import (
	"cmp"
	"encoding/hex"
	"net/url"
	"strings"
)

// IsValidRelayURL checks if a URL is a valid relay URL (ws:// or wss://).
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

// IsValid32ByteHex checks if a string is a valid 32-byte hex string.
func IsValid32ByteHex(thing string) bool {
	if !isLowerHex(thing) {
		return false
	}
	if len(thing) != 64 {
		return false
	}
	_, err := hex.DecodeString(thing)
	return err == nil
}

// CompareEvent is meant to to be used with slices.Sort
func CompareEvent(a, b Event) int {
	if a.CreatedAt == b.CreatedAt {
		return strings.Compare(a.ID, b.ID)
	}
	return cmp.Compare(a.CreatedAt, b.CreatedAt)
}

// CompareEventReverse is meant to to be used with slices.Sort
func CompareEventReverse(b, a Event) int {
	if a.CreatedAt == b.CreatedAt {
		return strings.Compare(a.ID, b.ID)
	}
	return cmp.Compare(a.CreatedAt, b.CreatedAt)
}

// CompareEventPtr is meant to to be used with slices.Sort
func CompareEventPtr(a, b *Event) int {
	if a == nil {
		if b == nil {
			return 0
		} else {
			return -1
		}
	} else if b == nil {
		return 1
	}

	if a.CreatedAt == b.CreatedAt {
		return strings.Compare(a.ID, b.ID)
	}
	return cmp.Compare(a.CreatedAt, b.CreatedAt)
}

// CompareEventPtrReverse is meant to to be used with slices.Sort
func CompareEventPtrReverse(b, a *Event) int {
	if a == nil {
		if b == nil {
			return 0
		} else {
			return -1
		}
	} else if b == nil {
		return 1
	}

	if a.CreatedAt == b.CreatedAt {
		return strings.Compare(a.ID, b.ID)
	}
	return cmp.Compare(a.CreatedAt, b.CreatedAt)
}
