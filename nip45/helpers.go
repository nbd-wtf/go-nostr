package nip45

import "math"

const two32 = 1 << 32

// Extract bits from uint64 using LSB 0 numbering, including lo.
func eb(bits uint64, hi uint8, lo uint8) uint64 {
	m := uint64(((1 << (hi - lo)) - 1) << lo)
	return (bits & m) >> lo
}

func linearCounting(m uint32, v uint32) float64 {
	fm := float64(m)
	return fm * math.Log(fm/float64(v))
}

func clz64(x uint64) uint8 {
	var c uint8
	for m := uint64(1 << 63); m&x == 0 && m != 0; m >>= 1 {
		c++
	}
	return c
}

func countZeros(s []uint8) uint32 {
	var c uint32
	for _, v := range s {
		if v == 0 {
			c++
		}
	}
	return c
}

func calculateEstimate(s []uint8) float64 {
	sum := 0.0
	for _, val := range s {
		sum += 1.0 / float64(uint64(1)<<val)
	}

	m := uint32(len(s))
	fm := float64(m)
	return alpha(m) * fm * fm / sum
}

func alpha(m uint32) float64 {
	if m == 16 {
		return 0.673
	} else if m == 32 {
		return 0.697
	} else if m == 64 {
		return 0.709
	}
	return 0.7213 / (1 + 1.079/float64(m))
}
