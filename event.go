package nostr

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"github.com/mailru/easyjson"
)

// Event represents a Nostr event.
type Event struct {
	ID        string
	PubKey    string
	CreatedAt Timestamp
	Kind      int
	Tags      Tags
	Content   string
	Sig       string
}

func (evt Event) String() string {
	j, _ := easyjson.Marshal(evt)
	return string(j)
}

// GetID computes the event ID and returns it as a hex string.
func (evt *Event) GetID() string {
	h := sha256.Sum256(evt.Serialize())
	return hex.EncodeToString(h[:])
}

// CheckID checks if the implied ID matches the given ID more efficiently.
func (evt *Event) CheckID() bool {
	if len(evt.ID) != 64 {
		return false
	}

	ser := make([]byte, 0, 100+len(evt.Content)+len(evt.Tags)*80)
	ser = serializeEventInto(evt, ser)
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

// Serialize outputs a byte array that can be hashed to produce the canonical event "id".
func (evt *Event) Serialize() []byte {
	// the serialization process is just putting everything into a JSON array
	// so the order is kept. See NIP-01
	dst := make([]byte, 0, 100+len(evt.Content)+len(evt.Tags)*80)
	return serializeEventInto(evt, dst)
}

func serializeEventInto(evt *Event, dst []byte) []byte {
	// the header portion is easy to serialize
	// [0,"pubkey",created_at,kind,[
	dst = append(dst, "[0,\""...)
	dst = append(dst, evt.PubKey...)
	dst = append(dst, "\","...)
	dst = append(dst, strconv.FormatInt(int64(evt.CreatedAt), 10)...)
	dst = append(dst, ',')
	dst = append(dst, strconv.Itoa(evt.Kind)...)
	dst = append(dst, ',')

	// tags
	dst = append(dst, '[')
	for i, tag := range evt.Tags {
		if i > 0 {
			dst = append(dst, ',')
		}
		// tag item
		dst = append(dst, '[')
		for i, s := range tag {
			if i > 0 {
				dst = append(dst, ',')
			}
			dst = escapeString(dst, s)
		}
		dst = append(dst, ']')
	}
	dst = append(dst, "],"...)

	// content needs to be escaped in general as it is user generated.
	dst = escapeString(dst, evt.Content)
	dst = append(dst, ']')

	return dst
}
