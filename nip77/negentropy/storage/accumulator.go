package storage

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"github.com/nbd-wtf/go-nostr/nip77/negentropy"
)

type Accumulator struct {
	Buf [32 + 8]byte // leave 8 bytes at the end as a slack for use in GetFingerprint append()
}

func (acc *Accumulator) Reset() {
	for i := 0; i < 32; i++ {
		acc.Buf[i] = 0
	}
}

func (acc *Accumulator) AddAccumulator(other Accumulator) {
	acc.AddBytes(other.Buf[:32])
}

func (acc *Accumulator) AddBytes(other []byte) {
	var currCarry, nextCarry uint32

	for i := 0; i < 8; i++ {
		offset := i * 4
		orig := binary.LittleEndian.Uint32(acc.Buf[offset:])
		otherV := binary.LittleEndian.Uint32(other[offset:])

		next := orig + currCarry + otherV
		if next < orig || next < otherV {
			nextCarry = 1
		}

		binary.LittleEndian.PutUint32(acc.Buf[offset:32], next&0xFFFFFFFF)
		currCarry = nextCarry
		nextCarry = 0
	}
}

func (acc *Accumulator) GetFingerprint(n int) string {
	input := acc.Buf[:32]
	input = append(input, negentropy.EncodeVarInt(n)...)
	hash := sha256.Sum256(input)
	return hex.EncodeToString(hash[:negentropy.FingerprintSize])
}
