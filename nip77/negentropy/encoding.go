package negentropy

import (
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

func (n *Negentropy) DecodeTimestampIn(reader *StringHexReader) (nostr.Timestamp, error) {
	t, err := decodeVarInt(reader)
	if err != nil {
		return 0, err
	}

	timestamp := nostr.Timestamp(t)
	if timestamp == 0 {
		timestamp = maxTimestamp
	} else {
		timestamp--
	}

	timestamp += n.lastTimestampIn
	if timestamp < n.lastTimestampIn { // Check for overflow
		timestamp = maxTimestamp
	}
	n.lastTimestampIn = timestamp
	return timestamp, nil
}

func (n *Negentropy) DecodeBound(reader *StringHexReader) (Bound, error) {
	timestamp, err := n.DecodeTimestampIn(reader)
	if err != nil {
		return Bound{}, fmt.Errorf("failed to decode bound timestamp: %w", err)
	}

	length, err := decodeVarInt(reader)
	if err != nil {
		return Bound{}, fmt.Errorf("failed to decode bound length: %w", err)
	}

	id, err := reader.ReadString(length * 2)
	if err != nil {
		return Bound{}, fmt.Errorf("failed to read bound id: %w", err)
	}

	return Bound{Item{timestamp, id}}, nil
}

func (n *Negentropy) encodeTimestampOut(w *StringHexWriter, timestamp nostr.Timestamp) {
	if timestamp == maxTimestamp {
		n.lastTimestampOut = maxTimestamp
		encodeVarIntToHex(w, 0)
		return
	}
	temp := timestamp
	timestamp -= n.lastTimestampOut
	n.lastTimestampOut = temp
	encodeVarIntToHex(w, int(timestamp+1))
	return
}

func (n *Negentropy) encodeBound(w *StringHexWriter, bound Bound) {
	n.encodeTimestampOut(w, bound.Timestamp)
	encodeVarIntToHex(w, len(bound.ID)/2)
	w.WriteHex(bound.Item.ID)
}

func getMinimalBound(prev, curr Item) Bound {
	if curr.Timestamp != prev.Timestamp {
		return Bound{Item{curr.Timestamp, ""}}
	}

	sharedPrefixBytes := 0

	for i := 0; i < 32; i++ {
		if curr.ID[i:i+2] != prev.ID[i:i+2] {
			break
		}
		sharedPrefixBytes++
	}

	// sharedPrefixBytes + 1 to include the first differing byte, or the entire ID if identical.
	return Bound{Item{curr.Timestamp, curr.ID[:(sharedPrefixBytes+1)*2]}}
}

func decodeVarInt(reader *StringHexReader) (int, error) {
	var res int = 0

	for {
		b, err := reader.ReadHexByte()
		if err != nil {
			return 0, err
		}

		res = (res << 7) | (int(b) & 127)
		if (b & 128) == 0 {
			break
		}
	}

	return res, nil
}

func encodeVarInt(n int) []byte {
	if n == 0 {
		return []byte{0}
	}

	var o []byte
	for n != 0 {
		o = append([]byte{byte(n & 0x7F)}, o...)
		n >>= 7
	}

	for i := 0; i < len(o)-1; i++ {
		o[i] |= 0x80
	}

	return o
}

func encodeVarIntToHex(w *StringHexWriter, n int) {
	if n == 0 {
		w.WriteByte(0)
	}

	var o []byte
	for n != 0 {
		o = append([]byte{byte(n & 0x7F)}, o...)
		n >>= 7
	}

	for i := 0; i < len(o)-1; i++ {
		o[i] |= 0x80
	}

	w.WriteBytes(o)
}
