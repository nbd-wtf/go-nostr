package test

import (
	"testing"

	"github.com/nbd-wtf/go-nostr/sdk/hints/memoryh"
)

func TestMemoryHints(t *testing.T) {
	runTestWith(t, memoryh.NewHintDB())
}
