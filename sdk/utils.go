package sdk

import (
	"math"
	"strings"
	"testing"
)

// IsVirtualRelay returns true if the given normalized relay URL shouldn't be considered for outbox-model calculations.
func IsVirtualRelay(url string) bool {
	if len(url) < 6 {
		// this is just invalid
		return true
	}

	if strings.HasPrefix(url, "wss://feeds.nostr.band") ||
		strings.HasPrefix(url, "wss://filter.nostr.wine") ||
		strings.HasPrefix(url, "wss://cache") {
		return true
	}

	if !testing.Testing() &&
		strings.HasPrefix(url, "ws://localhost") ||
		strings.HasPrefix(url, "ws://127.0.0.1") {
		return true
	}

	return false
}

// PerQueryLimitInBatch tries to make an educated guess for the batch size given the total filter limit and
// the number of abstract queries we'll be conducting at the same time.
func PerQueryLimitInBatch(totalFilterLimit int, numberOfQueries int) int {
	if numberOfQueries == 1 || totalFilterLimit*numberOfQueries < 50 {
		return totalFilterLimit
	}

	return max(4,
		int(
			math.Ceil(
				float64(totalFilterLimit)/
					math.Pow(float64(numberOfQueries), 0.4),
			),
		),
	)
}
