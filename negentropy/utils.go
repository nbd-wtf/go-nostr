package negentropy

import "bytes"

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
