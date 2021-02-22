package event

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/fiatjaf/bip340"
)

const (
	KindSetMetadata            uint8 = 0
	KindTextNote               uint8 = 1
	KindRecommendServer        uint8 = 2
	KindContactList            uint8 = 3
	KindEncryptedDirectMessage uint8 = 4
)

type Event struct {
	ID string `json:"id"` // it's the hash of the serialized event

	PubKey    string `json:"pubkey"`
	CreatedAt uint32 `json:"created_at"`

	Kind uint8 `json:"kind"`

	Tags    Tags   `json:"tags"`
	Content string `json:"content"`
	Sig     string `json:"sig"`
}

type Tags []Tag

func (t *Tags) Scan(src interface{}) error {
	var jtags []byte = make([]byte, 0)

	switch v := src.(type) {
	case []byte:
		jtags = v
	case string:
		jtags = []byte(v)
	default:
		return errors.New("couldn't scan tags, it's not a json string")
	}

	json.Unmarshal(jtags, &t)
	return nil
}

type Tag []interface{}

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
	arr[2] = int64(evt.CreatedAt)

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
	pubkeyb, err := hex.DecodeString(evt.PubKey)
	if err != nil {
		return false, err
	}
	if len(pubkeyb) != 32 {
		return false, fmt.Errorf("pubkey must be 32 bytes, not %d", len(pubkeyb))
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

	sig, err := hex.DecodeString(evt.Sig)
	if err != nil {
		return false, fmt.Errorf("signature is invalid hex: %w", err)
	}
	if len(sig) != 64 {
		return false, fmt.Errorf("signature must be 64 bytes, not %d", len(sig))
	}

	var p [32]byte
	copy(p[:], pubkeyb)

	var s [64]byte
	copy(s[:], sig)

	h := sha256.Sum256(evt.Serialize())

	return bip340.Verify(p, h, s)
}

// Sign signs an event with a given privateKey
func (evt *Event) Sign(privateKey string) error {
	h := sha256.Sum256(evt.Serialize())
	s, _ := new(big.Int).SetString(privateKey, 16)

	if s == nil {
		return errors.New("invalid private key " + privateKey)
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
