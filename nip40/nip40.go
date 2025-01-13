package nip40

import (
	"strconv"

	"github.com/nbd-wtf/go-nostr"
)

// GetExpiration returns the expiration timestamp for this event, or -1 if no "expiration" tag exists or
// if it is invalid.
func GetExpiration(tags nostr.Tags) nostr.Timestamp {
	for _, tag := range tags {
		if len(tag) >= 2 && tag[0] == "expiration" {
			if ts, err := strconv.ParseInt(tag[1], 10, 64); err == nil {
				return nostr.Timestamp(ts)
			} else {
				return -1
			}
		}
	}
	return -1
}
