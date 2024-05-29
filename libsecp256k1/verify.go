package libsecp256k1

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

func CheckSignature(evt *nostr.Event) (bool, error) {
	var pk [32]byte
	_, err := hex.Decode(pk[:], []byte(evt.PubKey))
	if err != nil {
		return false, fmt.Errorf("event pubkey '%s' is invalid hex: %w", evt.PubKey, err)
	}

	var sig [64]byte
	_, err = hex.Decode(sig[:], []byte(evt.Sig))
	if err != nil {
		return false, fmt.Errorf("event signature '%s' is invalid hex: %w", evt.Sig, err)
	}

	msg := sha256.Sum256(evt.Serialize())
	return Verify(msg, sig, pk), nil
}
