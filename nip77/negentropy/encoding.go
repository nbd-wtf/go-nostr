package negentropy

import (
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

func (n *Negentropy) readTimestamp(reader *StringHexReader) (nostr.Timestamp, error) {
	delta, err := readVarInt(reader)
	if err != nil {
		return 0, err
	}

	if delta == 0 {
		// zeroes are infinite
		timestamp := maxTimestamp
		n.lastTimestampIn = timestamp
		return timestamp, nil
	}

	// remove 1 as we always add 1 when encoding
	delta--

	// we add the previously cached timestamp to get the current
	timestamp := n.lastTimestampIn + nostr.Timestamp(delta)

	// cache this so we can apply it to the delta next time
	n.lastTimestampIn = timestamp

	return timestamp, nil
}

func (n *Negentropy) readBound(reader *StringHexReader) (Bound, error) {
	timestamp, err := n.readTimestamp(reader)
	if err != nil {
		return Bound{}, fmt.Errorf("failed to decode bound timestamp: %w", err)
	}

	length, err := readVarInt(reader)
	if err != nil {
		return Bound{}, fmt.Errorf("failed to decode bound length: %w", err)
	}

	id, err := reader.ReadString(length * 2)
	if err != nil {
		return Bound{}, fmt.Errorf("failed to read bound id: %w", err)
	}

	return Bound{Item{timestamp, id}}, nil
}

func (n *Negentropy) writeTimestamp(w *StringHexWriter, timestamp nostr.Timestamp) {
	if timestamp == maxTimestamp {
		// zeroes are infinite
		n.lastTimestampOut = maxTimestamp // cache this (see below)
		writeVarInt(w, 0)
		return
	}

	// we will only encode the difference between this timestamp and the previous
	delta := timestamp - n.lastTimestampOut

	// we cache this here as the next timestamp we encode will be just a delta from this
	n.lastTimestampOut = timestamp

	// add 1 to prevent zeroes from being read as infinites
	writeVarInt(w, int(delta+1))
	return
}

func (n *Negentropy) writeBound(w *StringHexWriter, bound Bound) {
	n.writeTimestamp(w, bound.Timestamp)
	writeVarInt(w, len(bound.ID)/2)
	w.WriteHex(bound.Item.ID)
}

func getMinimalBound(prev, curr Item) Bound {
	if curr.Timestamp != prev.Timestamp {
		return Bound{Item{curr.Timestamp, ""}}
	}

	sharedPrefixBytes := 0

	for i := 0; i < 32; i += 2 {
		if curr.ID[i:i+2] != prev.ID[i:i+2] {
			break
		}
		sharedPrefixBytes++
	}

	// sharedPrefixBytes + 1 to include the first differing byte, or the entire ID if identical.
	return Bound{Item{curr.Timestamp, curr.ID[:(sharedPrefixBytes+1)*2]}}
}

func readVarInt(reader *StringHexReader) (int, error) {
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

func writeVarInt(w *StringHexWriter, n int) {
	if n == 0 {
		w.WriteByte(0)
		return
	}

	w.WriteBytes(EncodeVarInt(n))
}

func EncodeVarInt(n int) []byte {
	if n == 0 {
		return []byte{0}
	}

	result := make([]byte, 8)
	idx := 7

	for n != 0 {
		result[idx] = byte(n & 0x7F)
		n >>= 7
		idx--
	}

	result = result[idx+1:]
	for i := 0; i < len(result)-1; i++ {
		result[i] |= 0x80
	}

	return result
}
