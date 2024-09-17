package test

import (
	"testing"

	"github.com/nbd-wtf/go-nostr/sdk/hints/memory"
)

func TestMemoryHints(t *testing.T) {
	runTestWith(t, memory.NewHintDB())
}
