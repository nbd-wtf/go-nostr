package nip45

import (
	"encoding/binary"
	"encoding/hex"
)

// Everything is hardcoded to use precision 8, i.e. 256 registers.
type HyperLogLog struct {
	registers []uint8
}

func New() *HyperLogLog {
	// precision is always 8
	// the number of registers is always 256 (1<<8)
	hll := &HyperLogLog{}
	hll.registers = make([]uint8, 256)
	return hll
}

func (hll *HyperLogLog) Encode() string {
	return hex.EncodeToString(hll.registers)
}

func (hll *HyperLogLog) Decode(enc string) error {
	_, err := hex.Decode(hll.registers, []byte(enc))
	return err
}

func (hll *HyperLogLog) Clear() {
	for i := range hll.registers {
		hll.registers[i] = 0
	}
}

func (hll *HyperLogLog) Add(id string) {
	x, _ := hex.DecodeString(id[32 : 32+8*2])
	j := x[0] // register address (first 8 bits, i.e. first byte)

	w := binary.BigEndian.Uint64(x) // number that we will use
	zeroBits := clz56(w) + 1        // count zeroes (skip the first byte, so only use 56 bits)

	if zeroBits > hll.registers[j] {
		hll.registers[j] = zeroBits
	}
}

func (hll *HyperLogLog) Merge(other *HyperLogLog) error {
	for i, v := range other.registers {
		if v > hll.registers[i] {
			hll.registers[i] = v
		}
	}
	return nil
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
