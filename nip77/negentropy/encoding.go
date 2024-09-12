package negentropy

import (
	"bytes"
	"encoding/hex"

	"github.com/nbd-wtf/go-nostr"
)

func (n *Negentropy) DecodeTimestampIn(reader *bytes.Reader) (nostr.Timestamp, error) {
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

func (n *Negentropy) DecodeBound(reader *bytes.Reader) (Bound, error) {
	timestamp, err := n.DecodeTimestampIn(reader)
	if err != nil {
		return Bound{}, err
	}

	length, err := decodeVarInt(reader)
	if err != nil {
		return Bound{}, err
	}

	id := make([]byte, length)
	if _, err = reader.Read(id); err != nil {
		return Bound{}, err
	}

	return Bound{Item{timestamp, hex.EncodeToString(id)}}, nil
}

func (n *Negentropy) encodeTimestampOut(timestamp nostr.Timestamp) []byte {
	if timestamp == maxTimestamp {
		n.lastTimestampOut = maxTimestamp
		return encodeVarInt(0)
	}
	temp := timestamp
	timestamp -= n.lastTimestampOut
	n.lastTimestampOut = temp
	return encodeVarInt(int(timestamp + 1))
}

func (n *Negentropy) encodeBound(bound Bound) []byte {
	var output []byte

	t := n.encodeTimestampOut(bound.Timestamp)
	idlen := encodeVarInt(len(bound.ID) / 2)
	output = append(output, t...)
	output = append(output, idlen...)
	id, _ := hex.DecodeString(bound.Item.ID)

	output = append(output, id...)
	return output
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
	return Bound{Item{curr.Timestamp, curr.ID[:sharedPrefixBytes*2+1]}}
}

func decodeVarInt(reader *bytes.Reader) (int, error) {
	var res int = 0

	for {
		b, err := reader.ReadByte()
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
