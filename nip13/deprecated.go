package nip13

import (
	"strconv"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

// Deprecated: use DoWork() instead.
func Generate(event *nostr.Event, targetDifficulty int, timeout time.Duration) (*nostr.Event, error) {
	if event.PubKey == "" {
		return nil, ErrMissingPubKey
	}

	tag := nostr.Tag{"nonce", "", strconv.Itoa(targetDifficulty)}
	event.Tags = append(event.Tags, tag)
	var nonce uint64
	start := time.Now()
	for {
		nonce++
		tag[1] = strconv.FormatUint(nonce, 10)
		if Difficulty(event.GetID()) >= targetDifficulty {
			return event, nil
		}
		// benchmarks show one iteration is approx 3000ns on i7-8565U @ 1.8GHz.
		// so, check every 30ms; arbitrary
		if nonce%10000 == 0 && time.Since(start) > timeout {
			return nil, ErrGenerateTimeout
		}
	}
}
