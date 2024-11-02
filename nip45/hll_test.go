package nip45

import (
	"encoding/hex"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHyperLogLog(t *testing.T) {
	rand := rand.New(rand.NewPCG(1, 0))

	for _, count := range []int{
		2, 4, 6, 7, 12, 15, 22, 36, 44, 47,
		64, 77, 89, 95, 104, 116, 122, 144,
		150, 199, 300, 350, 400, 500, 600,
		777, 922, 1000, 1500, 2222, 9999,
		13600, 80000, 133333, 200000,
	} {
		hll, _ := New(8)

		for range count {
			b := make([]byte, 32)
			for i := range b {
				b[i] = uint8(rand.UintN(256))
			}
			id := hex.EncodeToString(b)
			hll.Add(id)
		}

		res100 := int(hll.Count() * 100)
		require.Greater(t, res100, count*85, "result too low (actual %d < %d)", hll.Count(), count)
		require.Less(t, res100, count*115, "result too high (actual %d > %d)", hll.Count(), count)
	}
}

func TestHyperLogLogMerge(t *testing.T) {
	rand := rand.New(rand.NewPCG(2, 0))

	for _, count := range []int{
		2, 4, 6, 7, 12, 15, 22, 36, 44, 47,
		64, 77, 89, 95, 104, 116, 122, 144,
		150, 199, 300, 350, 400, 500, 600,
		777, 922, 1000, 1500, 2222, 9999,
		13600, 80000, 133333, 200000,
	} {
		hllA, _ := New(8)
		hllB, _ := New(8)

		for range count / 2 {
			b := make([]byte, 32)
			for i := range b {
				b[i] = uint8(rand.UintN(256))
			}
			id := hex.EncodeToString(b)
			hllA.Add(id)
		}
		for range count / 2 {
			b := make([]byte, 32)
			for i := range b {
				b[i] = uint8(rand.UintN(256))
			}
			id := hex.EncodeToString(b)
			hllB.Add(id)
		}

		hll, _ := New(8)
		hll.Merge(hllA)
		hll.Merge(hllB)

		res100 := int(hll.Count() * 100)
		require.Greater(t, res100, count*85, "result too low (actual %d < %d)", hll.Count(), count)
		require.Less(t, res100, count*115, "result too high (actual %d > %d)", hll.Count(), count)
	}
}

func TestHyperLogLogMergeComplex(t *testing.T) {
	rand := rand.New(rand.NewPCG(2, 0))

	for _, count := range []int{
		3, 6, 9, 12, 15, 22, 36, 46, 57,
		64, 77, 89, 95, 104, 116, 122, 144,
		150, 199, 300, 350, 400, 500, 600,
		777, 922, 1000, 1500, 2222, 9999,
		13600, 80000, 133333, 200000,
	} {
		hllA, _ := New(8)
		hllB, _ := New(8)
		hllC, _ := New(8)

		for range count / 3 {
			b := make([]byte, 32)
			for i := range b {
				b[i] = uint8(rand.UintN(256))
			}
			id := hex.EncodeToString(b)
			hllA.Add(id)
			hllC.Add(id)
		}
		for range count / 3 {
			b := make([]byte, 32)
			for i := range b {
				b[i] = uint8(rand.UintN(256))
			}
			id := hex.EncodeToString(b)
			hllB.Add(id)
			hllC.Add(id)
		}
		for range count / 3 {
			b := make([]byte, 32)
			for i := range b {
				b[i] = uint8(rand.UintN(256))
			}
			id := hex.EncodeToString(b)
			hllC.Add(id)
			hllA.Add(id)
		}

		hll, _ := New(8)
		hll.Merge(hllA)
		hll.Merge(hllB)
		hll.Merge(hllC)

		res100 := int(hll.Count() * 100)
		require.Greater(t, res100, count*85, "result too low (actual %d < %d)", hll.Count(), count)
		require.Less(t, res100, count*115, "result too high (actual %d > %d)", hll.Count(), count)
	}
}
