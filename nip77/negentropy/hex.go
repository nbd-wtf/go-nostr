package negentropy

import (
	"encoding/hex"
	"io"
)

func NewStringHexReader(source string) *StringHexReader {
	return &StringHexReader{source, 0, make([]byte, 1)}
}

type StringHexReader struct {
	source string
	idx    int

	tmp []byte
}

func (r *StringHexReader) Len() int {
	return len(r.source) - r.idx
}

func (r *StringHexReader) ReadHexBytes(buf []byte) error {
	n := len(buf) * 2
	r.idx += n
	if len(r.source) < r.idx {
		return io.EOF
	}
	_, err := hex.Decode(buf, []byte(r.source[r.idx-n:r.idx]))
	return err
}

func (r *StringHexReader) ReadHexByte() (byte, error) {
	err := r.ReadHexBytes(r.tmp)
	return r.tmp[0], err
}

func (r *StringHexReader) ReadString(size int) (string, error) {
	r.idx += size
	if len(r.source) < r.idx {
		return "", io.EOF
	}
	return r.source[r.idx-size : r.idx], nil
}

func NewStringHexWriter(buf []byte) *StringHexWriter {
	return &StringHexWriter{buf, make([]byte, 2)}
}

type StringHexWriter struct {
	hexbuf []byte

	tmp []byte
}

func (r *StringHexWriter) Len() int {
	return len(r.hexbuf)
}

func (r *StringHexWriter) Hex() string {
	return string(r.hexbuf)
}

func (r *StringHexWriter) Reset() {
	r.hexbuf = r.hexbuf[:0]
}

func (r *StringHexWriter) WriteHex(hexString string) {
	r.hexbuf = append(r.hexbuf, hexString...)
	return
}

func (r *StringHexWriter) WriteByte(b byte) error {
	hex.Encode(r.tmp, []byte{b})
	r.hexbuf = append(r.hexbuf, r.tmp...)
	return nil
}

func (r *StringHexWriter) WriteBytes(in []byte) {
	r.hexbuf = hex.AppendEncode(r.hexbuf, in)

	// curr := len(r.hexbuf)
	// next := curr + len(in)*2
	// for cap(r.hexbuf) < next {
	// 	r.hexbuf = append(r.hexbuf, in...)
	// }
	// r.hexbuf = r.hexbuf[0:next]
	// dst := r.hexbuf[curr:next]

	// hex.Encode(dst, in)

	return
}
