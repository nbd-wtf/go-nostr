package negentropy

import (
	"cmp"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"iter"
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

const FingerprintSize = 16

type Mode uint8

const (
	SkipMode        Mode = 0
	FingerprintMode Mode = 1
	IdListMode      Mode = 2
)

func (v Mode) String() string {
	switch v {
	case SkipMode:
		return "SKIP"
	case FingerprintMode:
		return "FINGERPRINT"
	case IdListMode:
		return "IDLIST"
	default:
		return "<UNKNOWN-ERROR>"
	}
}

type Storage interface {
	Insert(nostr.Timestamp, string) error
	Seal()
	Size() int
	Range(begin, end int) iter.Seq2[int, Item]
	FindLowerBound(begin, end int, value Bound) int
	GetBound(idx int) Bound
	Fingerprint(begin, end int) [FingerprintSize]byte
}

type Item struct {
	Timestamp nostr.Timestamp
	ID        string
}

func itemCompare(a, b Item) int {
	if a.Timestamp == b.Timestamp {
		return strings.Compare(a.ID, b.ID)
	}
	return cmp.Compare(a.Timestamp, b.Timestamp)
}

func (i Item) String() string { return fmt.Sprintf("Item<%d:%s>", i.Timestamp, i.ID) }

type Bound struct{ Item }

func (b Bound) String() string {
	if b.Timestamp == infiniteBound.Timestamp {
		return "Bound<infinite>"
	}
	return fmt.Sprintf("Bound<%d:%s>", b.Timestamp, b.ID)
}

type Accumulator struct {
	Buf []byte
}

func (acc *Accumulator) SetToZero() {
	acc.Buf = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
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

func (acc *Accumulator) GetFingerprint(n int) [FingerprintSize]byte {
	input := acc.Buf[:]
	input = append(input, encodeVarInt(n)...)

	hash := sha256.Sum256(input)

	var fingerprint [FingerprintSize]byte
	copy(fingerprint[:], hash[:FingerprintSize])
	return fingerprint
}
