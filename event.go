package nostr

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/mailru/easyjson"
)

type Event struct {
	ID        string
	PubKey    string
	CreatedAt Timestamp
	Kind      int
	Tags      Tags
	Content   string
	Sig       string

	// anything here will be mashed together with the main event object when serializing
	extra map[string]any
}

// Event Stringer interface, just returns the raw JSON as a string.
func (evt Event) String() string {
	j, _ := easyjson.Marshal(evt)
	return string(j)
}

// GetID serializes and returns the event ID as a string.
func (evt *Event) GetID() string {
	h := sha256.Sum256(evt.Serialize())
	return hex.EncodeToString(h[:])
}

// CheckID checks if the implied ID matches the given ID
func (evt *Event) CheckID() bool {
	ser := evt.Serialize()
	h := sha256.Sum256(ser)

	const hextable = "0123456789abcdef"

	for i := 0; i < 32; i++ {
		b := hextable[h[i]>>4]
		if b != evt.ID[i*2] {
			return false
		}

		b = hextable[h[i]&0x0f]
		if b != evt.ID[i*2+1] {
			return false
		}
	}

	return true
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
			evt.CreatedAt,
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

// Sign signs an event with a given privateKey.
func (evt *Event) Sign(privateKey string, signOpts ...schnorr.SignOption) error {
	s, err := hex.DecodeString(privateKey)
	if err != nil {
		return fmt.Errorf("Sign called with invalid private key '%s': %w", privateKey, err)
	}

	if evt.Tags == nil {
		evt.Tags = make(Tags, 0)
	}

	sk, pk := btcec.PrivKeyFromBytes(s)
	pkBytes := pk.SerializeCompressed()
	evt.PubKey = hex.EncodeToString(pkBytes[1:])

	h := sha256.Sum256(evt.Serialize())
	sig, err := schnorr.Sign(sk, h[:], signOpts...)
	if err != nil {
		return err
	}

	evt.ID = hex.EncodeToString(h[:])
	evt.Sig = hex.EncodeToString(sig.Serialize())

	return nil
}

// IsRegular checks if the given kind is in Regular range.
func (evt *Event) IsRegular() bool {
	return evt.Kind < 10000 && evt.Kind != 0 && evt.Kind != 3
}

// IsReplaceable checks if the given kind is in Replaceable range.
func (evt *Event) IsReplaceable() bool {
	return evt.Kind == 0 || evt.Kind == 3 ||
		(10000 <= evt.Kind && evt.Kind < 20000)
}

// IsEphemeral checks if the given kind is in Ephemeral range.
func (evt *Event) IsEphemeral() bool {
	return 20000 <= evt.Kind && evt.Kind < 30000
}

// IsAddressable checks if the given kind is in Addressable range.
func (evt *Event) IsAddressable() bool {
	return 30000 <= evt.Kind && evt.Kind < 40000
}
