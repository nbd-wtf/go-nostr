// Package nip13 implements NIP-13
// See https://github.com/nostr-protocol/nips/blob/master/13.md for details.
package nip13

import (
	"encoding/hex"
	"errors"
	"math/bits"
	"strconv"
	"time"

	nostr "github.com/nbd-wtf/go-nostr"
)

var (
	ErrDifficultyTooLow = errors.New("nip13: insufficient difficulty")
	ErrGenerateTimeout  = errors.New("nip13: generating proof of work took too long")
)

// Difficulty counts the number of leading zero bits in an event ID.
func Difficulty(id string) int {
	var zeros int
	var b [1]byte
	for i := 0; i < 64; i += 2 {
		if id[i:i+2] == "00" {
			zeros += 8
			continue
		}
		if _, err := hex.Decode(b[:], []byte{id[i], id[i+1]}); err != nil {
			return -1
		}
		zeros += bits.LeadingZeros8(b[0])
		break
	}
	return zeros
}

// Check reports whether the event ID demonstrates a sufficient proof of work difficulty.
// Note that Check performs no validation other than counting leading zero bits
// in an event ID. It is up to the callers to verify the event with other methods,
// such as [nostr.Event.CheckSignature].
func Check(id string, minDifficulty int) error {
	if Difficulty(id) < minDifficulty {
		return ErrDifficultyTooLow
	}
	return nil
}

// Generate performs proof of work on the specified event until either the target
// difficulty is reached or the function runs for longer than the timeout.
// The latter case results in ErrGenerateTimeout.
//
// Upon success, the returned event always contains a "nonce" tag with the target difficulty
// commitment, and an updated event.CreatedAt.
func Generate(event *nostr.Event, targetDifficulty int, timeout time.Duration) (*nostr.Event, error) {
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
