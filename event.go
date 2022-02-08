package nostr

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/fiatjaf/bip340"
)

const (
	KindSetMetadata            int = 0
	KindTextNote               int = 1
	KindRecommendServer        int = 2
	KindContactList            int = 3
	KindEncryptedDirectMessage int = 4
	KindDeletion               int = 5
)

type Event struct {
	ID string `json:"id"` // it's the hash of the serialized event

	PubKey    string `json:"pubkey"`
	CreatedAt Time   `json:"created_at"`

	Kind int `json:"kind"`

	Tags    Tags   `json:"tags"`
	Content string `json:"content"`
	Sig     string `json:"sig"`
}

type Time time.Time

func (tm *Time) UnmarshalJSON(payload []byte) error {
	var unix int64
	err := json.Unmarshal(payload, &unix)
	if err != nil {
		return fmt.Errorf("time must be a unix timestamp as an integer, not '%s': %w",
			string(payload), err)
	}
	t := Time(time.Unix(unix, 0))
	tm = &t
	return nil
}

func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(time.Time(t).Unix(), 10)), nil
}

// Serialize outputs a byte array that can be hashed/signed to identify/authenticate
func (evt *Event) Serialize() []byte {
	// the serialization process is just putting everything into a JSON array
	// so the order is kept
	arr := make([]interface{}, 6)

	// version: 0
	arr[0] = 0

	// pubkey
	arr[1] = evt.PubKey

	// created_at
	arr[2] = int64(time.Time(evt.CreatedAt).Unix())

	// kind
	arr[3] = int64(evt.Kind)

	// tags
	if evt.Tags != nil {
		arr[4] = evt.Tags
	} else {
		arr[4] = make([]bool, 0)
	}

	// content
	arr[5] = evt.Content

	serialized := new(bytes.Buffer)

	enc := json.NewEncoder(serialized)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(arr)
	return serialized.Bytes()[:serialized.Len()-1] // Encode add new line char
}

// CheckSignature checks if the signature is valid for the id
// (which is a hash of the serialized event content).
// returns an error if the signature itself is invalid.
func (evt Event) CheckSignature() (bool, error) {
	// read and check pubkey
	pubkey, err := bip340.ParsePublicKey(evt.PubKey)
	if err != nil {
		return false, fmt.Errorf("Event has invalid pubkey '%s': %w", evt.PubKey, err)
	}

	// check tags
	for _, tag := range evt.Tags {
		for _, item := range tag {
			switch item.(type) {
			case string, int64, float64, int, bool:
				// fine
			default:
				// not fine
				return false, fmt.Errorf("tag contains an invalid value %v", item)
			}
		}
	}

	s, err := hex.DecodeString(evt.Sig)
	if err != nil {
		return false, fmt.Errorf("signature is invalid hex: %w", err)
	}
	if len(s) != 64 {
		return false, fmt.Errorf("signature must be 64 bytes, not %d", len(s))
	}

	var sig [64]byte
	copy(sig[:], s)

	hash := sha256.Sum256(evt.Serialize())
	return bip340.Verify(pubkey, hash, sig)
}

// Sign signs an event with a given privateKey
func (evt *Event) Sign(privateKey string) error {
	h := sha256.Sum256(evt.Serialize())

	s, err := bip340.ParsePrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("Sign called with invalid private key '%s': %w", privateKey, err)
	}

	aux := make([]byte, 32)
	rand.Read(aux)
	sig, err := bip340.Sign(s, h, aux)
	if err != nil {
		return err
	}

	evt.ID = hex.EncodeToString(h[:])
	evt.Sig = hex.EncodeToString(sig[:])
	return nil
}
