package hyperloglog

import (
	"math"
)

const two32 = 1 << 32

func linearCounting(m uint32, v uint32) float64 {
	fm := float64(m)
	return fm * math.Log(fm/float64(v))
}

func clz56(x uint64) uint8 {
	var c uint8
	for m := uint64(1 << 55); m&x == 0 && m != 0; m >>= 1 {
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
