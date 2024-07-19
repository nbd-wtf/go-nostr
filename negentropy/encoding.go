package negentropy

import (
	"errors"
)

var ErrParseEndsPrematurely = errors.New("parse ends prematurely")

func getByte(encoded *[]byte) (byte, error) {
	if len(*encoded) < 1 {
		return 0, ErrParseEndsPrematurely
	}
	b := (*encoded)[0]
	*encoded = (*encoded)[1:]

	return b, nil
}

func getBytes(encoded *[]byte, n int) ([]byte, error) {
	//fmt.Fprintln(os.Stderr, "getBytes", len(*encoded), n)
	if len(*encoded) < n {
		return nil, errors.New("parse ends prematurely")
	}
	result := (*encoded)[:n]
	*encoded = (*encoded)[n:]
	return result, nil
}

func decodeVarInt(encoded *[]byte) (int, error) {
	//var res uint64
	//
	//for i := 0; i < len(*encoded); i++ {
	//	byte := (*encoded)[i]
	//	res = (res << 7) | uint64(byte&0x7F)
	//	if (byte & 0x80) == 0 {
	//		fmt.Fprintln(os.Stderr, "decodeVarInt", encoded, i)
	//		*encoded = (*encoded)[i+1:] // Advance the slice to reflect consumed bytes
	//		return res, nil
	//	}
	//}
	//return 0, ErrParseEndsPrematurely
	res := 0

	for {
		if len(*encoded) == 0 {
			return 0, errors.New("parse ends prematurely")
		}

		// Remove the first byte from the slice and update the slice.
		// This simulates JavaScript's shift operation on arrays.
		byte := (*encoded)[0]
		*encoded = (*encoded)[1:]

		res = (res << 7) | (int(byte) & 127)
		if (byte & 128) == 0 {
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