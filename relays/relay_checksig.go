//go:build !libsecp256k1

package relays

import "github.com/nbd-wtf/go-nostr/core"

func checkSigOnRelay(evt core.Event) bool {
	ok, _ := evt.CheckSignature()
	return ok
}
