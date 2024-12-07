package hyperloglog

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// Everything is hardcoded to use precision 8, i.e. 256 registers.
type HyperLogLog struct {
	offset    int
	registers []uint8
}

func New(offset int) *HyperLogLog {
	if offset < 0 || offset > 32-8 {
		panic(fmt.Errorf("invalid offset %d", offset))
	}

	// precision is always 8
	// the number of registers is always 256 (1<<8)
	hll := &HyperLogLog{offset: offset}
	hll.registers = make([]uint8, 256)
	return hll
}

func NewWithRegisters(registers []byte, offset int) *HyperLogLog {
	if offset < 0 || offset > 32-8 {
		panic(fmt.Errorf("invalid offset %d", offset))
	}
	if len(registers) != 256 {
		panic(fmt.Errorf("invalid number of registers %d", len(registers)))
	}
	return &HyperLogLog{registers: registers, offset: offset}
}

func (hll *HyperLogLog) GetRegisters() []byte    { return hll.registers }
func (hll *HyperLogLog) SetRegisters(enc []byte) { hll.registers = enc }
func (hll *HyperLogLog) MergeRegisters(other []byte) {
	for i, v := range other {
		if v > hll.registers[i] {
			hll.registers[i] = v
		}
	}
}

func (hll *HyperLogLog) Clear() {
	for i := range hll.registers {
		hll.registers[i] = 0
	}
}

// Add takes a Nostr event pubkey which will be used as the item "key" (that combined with the offset)
func (hll *HyperLogLog) Add(pubkey string) {
	x, _ := hex.DecodeString(pubkey[hll.offset*2 : hll.offset*2+8*2])
	j := x[0] // register address (first 8 bits, i.e. first byte)

	w := binary.BigEndian.Uint64(x) // number that we will use
	zeroBits := clz56(w) + 1        // count zeroes (skip the first byte, so only use 56 bits)

	if zeroBits > hll.registers[j] {
		hll.registers[j] = zeroBits
	}
}

// AddBytes is like Add, but takes pubkey as bytes instead of as string
func (hll *HyperLogLog) AddBytes(pubkey []byte) {
	x := pubkey[hll.offset : hll.offset+8]
	j := x[0] // register address (first 8 bits, i.e. first byte)

	w := binary.BigEndian.Uint64(x) // number that we will use
	zeroBits := clz56(w) + 1        // count zeroes (skip the first byte, so only use 56 bits)

	if zeroBits > hll.registers[j] {
		hll.registers[j] = zeroBits
	}
}

func (hll *HyperLogLog) Merge(other *HyperLogLog) {
	for i, v := range other.registers {
		if v > hll.registers[i] {
			hll.registers[i] = v
		}
	}
}

func (hll *HyperLogLog) Count() uint64 {
	v := countZeros(hll.registers)

	if v != 0 {
		lc := linearCounting(256 /* nregisters */, v)

		if lc <= 220 /* threshold */ {
			return uint64(lc)
		}
	}

	est := hll.calculateEstimate()
	if est <= 256 /* nregisters */ *3 {
		if v != 0 {
			return uint64(linearCounting(256 /* nregisters */, v))
		}
	}

	return uint64(est)
}

func (hll HyperLogLog) calculateEstimate() float64 {
	sum := 0.0
	for _, val := range hll.registers {
		sum += 1.0 / float64(uint64(1)<<val) // this is the same as 2^(-val)
	}

	return 0.7182725932495458 /* alpha for 256 registers */ * 256 /* nregisters */ * 256 /* nregisters */ / sum
}
