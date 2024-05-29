package libsecp256k1

import (
	"encoding/json"
	"testing"

	"github.com/nbd-wtf/go-nostr/core"
	"github.com/nbd-wtf/go-nostr/test_common"
)

func BenchmarkSignatureVerification(b *testing.B) {
	events := make([]*core.Event, len(test_common.NormalEvents))
	for i, jevt := range test_common.NormalEvents {
		evt := &core.Event{}
		json.Unmarshal([]byte(jevt), evt)
		events[i] = evt
	}

	b.Run("btcec", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, evt := range events {
				evt.CheckSignature()
			}
		}
	})

	b.Run("libsecp256k1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, evt := range events {
				CheckSignature(*evt)
			}
		}
	})
}
