package negentropy

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/big"
)

const (
	IDSize          = 32
	FingerprintSize = 16
)

var modulo = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)

var ErrBadIDSize = errors.New("bad id size")

type Mode int

const (
	SkipMode = iota
	FingerprintMode
	IdListMode
)

type Item struct {
	Timestamp uint64
	ID        string
}

func NewItem(timestamp uint64, id string) *Item {
	return &Item{Timestamp: timestamp, ID: id}
}

func (i Item) Equals(other Item) bool {
	return i.Timestamp == other.Timestamp && i.ID == other.ID
}

func (i Item) LessThan(other Item) bool {
	if i.Timestamp != other.Timestamp {
		return i.Timestamp < other.Timestamp
	}
	return i.ID < other.ID
}

type Bound struct {
	Item  Item
	IDLen int
}

// NewBound creates a new Bound instance with a timestamp and ID.
// It returns an error if the ID size is incorrect.
func NewBound(timestamp uint64, id string) (*Bound, error) {
	b := &Bound{
		Item:  *NewItem(timestamp, id),
		IDLen: len(id),
	}
	return b, nil
}

// NewBoundWithItem creates a new Bound instance from an existing Item.
func NewBoundWithItem(item Item) *Bound {
	return &Bound{
		Item:  item,
		IDLen: len(item.ID),
	}
}

// Equals checks if two Bound instances are equal.
func (b Bound) Equals(other Bound) bool {
	return b.Item.Equals(other.Item)
}

func (b Bound) LessThan(other Bound) bool {
	return b.Item.LessThan(other.Item)
}

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

func (acc *Accumulator) AddItem(other Item) {
	b, _ := hex.DecodeString(other.ID)
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
