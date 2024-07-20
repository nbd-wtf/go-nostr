package negentropy

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"github.com/nbd-wtf/go-nostr"
)

const FingerprintSize = 16

type Mode int

const (
	SkipMode = iota
	FingerprintMode
	IdListMode
)

type Storage interface {
	Insert(nostr.Timestamp, string) error
	Seal() error

	IDSize() int
	Size() int
	Iterate(begin, end int, cb func(item Item, i int) bool) error
	FindLowerBound(begin, end int, value Bound) (int, error)
	Fingerprint(begin, end int) (Fingerprint, error)
}

type Item struct {
	Timestamp nostr.Timestamp
	ID        string
}

func (i Item) LessThan(other Item) bool {
	if i.Timestamp != other.Timestamp {
		return i.Timestamp < other.Timestamp
	}
	return i.ID < other.ID
}

type Bound struct{ Item }

type Fingerprint struct {
	Buf [FingerprintSize]byte
}

func (f *Fingerprint) SV() []byte {
	return f.Buf[:]
}

type Accumulator struct {
	Buf []byte
}

func (acc *Accumulator) SetToZero() {
	acc.Buf = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
}

func (acc *Accumulator) Add(id string) {
	b, _ := hex.DecodeString(id)
	acc.AddBytes(b)
}

func (acc *Accumulator) AddAccumulator(other Accumulator) {
	acc.AddBytes(other.Buf)
}

func (acc *Accumulator) AddBytes(other []byte) {
	var currCarry, nextCarry uint32

	if len(acc.Buf) < 32 {
		newBuf := make([]byte, 32)
		copy(newBuf, acc.Buf)
		acc.Buf = newBuf
	}

	for i := 0; i < 8; i++ {
		offset := i * 4
		orig := binary.LittleEndian.Uint32(acc.Buf[offset:])
		otherV := binary.LittleEndian.Uint32(other[offset:])

		next := orig + currCarry + otherV
		if next < orig || next < otherV {
			nextCarry = 1
		}

		binary.LittleEndian.PutUint32(acc.Buf[offset:], next&0xFFFFFFFF)
		currCarry = nextCarry
		nextCarry = 0
	}
}

func (acc *Accumulator) Negate() {
	for i := range acc.Buf {
		acc.Buf[i] = ^acc.Buf[i]
	}

	var one []byte
	one[0] = 1 // Assuming little-endian; if big-endian, use one[len(one)-1] = 1

	acc.AddBytes(one)
}

func (acc *Accumulator) SV() []byte {
	return acc.Buf[:]
}

func (acc *Accumulator) GetFingerprint(n int) Fingerprint {
	input := acc.SV()
	input = append(input, encodeVarInt(n)...)

	hash := sha256.Sum256(input)

	var fingerprint Fingerprint
	copy(fingerprint.Buf[:], hash[:FingerprintSize])
	return fingerprint
}
