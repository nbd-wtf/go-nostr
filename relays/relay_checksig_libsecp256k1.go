//go:build libsecp256k1

package relays

import (
	"github.com/nbd-wtf/go-nostr/core"
	"github.com/nbd-wtf/go-nostr/libsecp256k1"
)

func checkSigOnRelay(evt core.Event) bool {
	ok, _ := libsecp256k1.CheckSignature(evt)
	return ok
}
