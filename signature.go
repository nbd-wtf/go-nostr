//go:build !libsecp256k1

package nostr

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

// CheckSignature checks if the event signature is valid for the given event.
// It won't look at the ID field, instead it will recompute the id from the entire event body.
// If the signature is invalid bool will be false and err will be set.
func (evt Event) CheckSignature() (bool, error) {
	// read and check pubkey
	pk, err := hex.DecodeString(evt.PubKey)
	if err != nil {
		return false, fmt.Errorf("event pubkey '%s' is invalid hex: %w", evt.PubKey, err)
	}

	pubkey, err := schnorr.ParsePubKey(pk)
	if err != nil {
		return false, fmt.Errorf("event has invalid pubkey '%s': %w", evt.PubKey, err)
	}

	// read signature
	s, err := hex.DecodeString(evt.Sig)
	if err != nil {
		return false, fmt.Errorf("signature '%s' is invalid hex: %w", evt.Sig, err)
	}
	sig, err := schnorr.ParseSignature(s)
	if err != nil {
		return false, fmt.Errorf("failed to parse signature: %w", err)
	}

	// check signature
	hash := sha256.Sum256(evt.Serialize())
	return sig.Verify(hash[:], pubkey), nil
}

// Sign signs an event with a given privateKey.
// It sets the event's ID, PubKey, and Sig fields.
// Returns an error if the private key is invalid or if signing fails.
func (evt *Event) Sign(secretKey string) error {
	s, err := hex.DecodeString(secretKey)
	if err != nil {
		return fmt.Errorf("Sign called with invalid secret key '%s': %w", secretKey, err)
	}

	if evt.Tags == nil {
		evt.Tags = make(Tags, 0)
	}

	sk, pk := btcec.PrivKeyFromBytes(s)
	pkBytes := pk.SerializeCompressed()
	evt.PubKey = hex.EncodeToString(pkBytes[1:])

	h := sha256.Sum256(evt.Serialize())
	sig, err := schnorr.Sign(sk, h[:], schnorr.FastSign())
	if err != nil {
		return err
	}

	evt.ID = hex.EncodeToString(h[:])
	evt.Sig = hex.EncodeToString(sig.Serialize())

	return nil
}
