package sdk

import (
	"math"
	"strings"
	"sync"
	"time"
)

var (
	_dtnmtoah     map[string]time.Time
	_dtnmtoahLock sync.Mutex
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

	return false
}

// BatchSizePerNumberOfQueries tries to make an educated guess for the batch size given the total filter limit and
// the number of abstract queries we'll be conducting at the same time
func BatchSizePerNumberOfQueries(totalFilterLimit int, numberOfQueries int) int {
	if numberOfQueries == 1 || totalFilterLimit*numberOfQueries < 50 {
		return totalFilterLimit
	}

	return int(
		math.Ceil(
			math.Pow(float64(totalFilterLimit), 0.80) / math.Pow(float64(numberOfQueries), 0.71),
		),
	)
}
