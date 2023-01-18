package nostr

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"time"
)

type Event struct {
	ID        string
	PubKey    string
	CreatedAt time.Time
	Kind      int
	Tags      Tags
	Content   string
	Sig       string

	// anything here will be mashed together with the main event object when serializing
	extra map[string]any
}

const (
	KindSetMetadata            int = 0
	KindTextNote               int = 1
	KindRecommendServer        int = 2
	KindContactList            int = 3
	KindEncryptedDirectMessage int = 4
	KindDeletion               int = 5
	KindBoost                  int = 6
	KindReaction               int = 7
	KindChannelCreation        int = 40
	KindChannelMetadata        int = 41
	KindChannelMessage         int = 42
	KindChannelHideMessage     int = 43
	KindChannelMuteUser        int = 44
)

// GetID serializes and returns the event ID as a string
func (evt *Event) GetID() string {
	h := sha256.Sum256(evt.Serialize())
	return hex.EncodeToString(h[:])
}

// Serialize outputs a byte array that can be hashed/signed to identify/authenticate.
// JSON encoding as defined in RFC4627.
func (evt *Event) Serialize() []byte {
	// the serialization process is just putting everything into a JSON array
	// so the order is kept. See NIP-01
	dst := make([]byte, 0)

	// the header portion is easy to serialize
	// [0,"pubkey",created_at,kind,[
	dst = append(dst, []byte(
		fmt.Sprintf(
			"[0,\"%s\",%d,%d,",
			evt.PubKey,
			evt.CreatedAt.Unix(),
			evt.Kind,
		))...)

	// tags
	dst = evt.Tags.marshalTo(dst)
	dst = append(dst, ',')

	// content needs to be escaped in general as it is user generated.
	dst = escapeString(dst, evt.Content)
	dst = append(dst, ']')

	return dst
}

// CheckSignature checks if the signature is valid for the id
// (which is a hash of the serialized event content).
// returns an error if the signature itself is invalid.
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

// Sign signs an event with a given privateKey
func (evt *Event) Sign(privateKey string) error {
	h := sha256.Sum256(evt.Serialize())

	s, err := hex.DecodeString(privateKey)
	if err != nil {
		return fmt.Errorf("Sign called with invalid private key '%s': %w", privateKey, err)
	}
	sk, _ := btcec.PrivKeyFromBytes(s)

	sig, err := schnorr.Sign(sk, h[:])
	if err != nil {
		return err
	}

	evt.ID = hex.EncodeToString(h[:])
	evt.Sig = hex.EncodeToString(sig.Serialize())
	return nil
}
