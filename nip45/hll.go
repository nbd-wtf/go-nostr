package nip45

import (
	"fmt"
	"strconv"
)

var threshold = []uint{
	10, 20, 40, 80, 220, 400, 900, 1800, 3100,
	6500, 11500, 20000, 50000, 120000, 350000,
}

type HyperLogLog struct {
	registers []uint8
	precision uint8
}

func New(precision uint8) (*HyperLogLog, error) {
	if precision > 16 || precision < 4 {
		return nil, fmt.Errorf("precision must be between 4 and 16")
	}

	hll := &HyperLogLog{}
	hll.precision = precision
	hll.registers = make([]uint8, 1<<precision)
	return hll, nil
}

func (hll *HyperLogLog) Clear() {
	for i := range hll.registers {
		hll.registers[i] = 0
	}
}

func (hll *HyperLogLog) Add(id string) {
	x, _ := strconv.ParseUint(id[32:32+8*2], 16, 64)

	i := eb(x, 64, 64-hll.precision)             // {x31,...,x32-p}
	w := x<<hll.precision | 1<<(hll.precision-1) // {x32-p,...,x0}

	zeroBits := clz64(w) + 1
	if zeroBits > hll.registers[i] {
		hll.registers[i] = zeroBits
	}
}

func (hll *HyperLogLog) Merge(other *HyperLogLog) error {
	if hll.precision != other.precision {
		return fmt.Errorf("precisions must be equal")
	}

	for i, v := range other.registers {
		if v > hll.registers[i] {
			hll.registers[i] = v
		}
	}

	return nil
}

func (hll *HyperLogLog) Count() uint64 {
	m := uint32(len(hll.registers))

	if v := countZeros(hll.registers); v != 0 {
		lc := linearCounting(m, v)
		if lc <= float64(threshold[hll.precision-4]) {
			return uint64(lc)
		}
	}

	est := calculateEstimate(hll.registers)
	if est <= float64(len(hll.registers))*5.0 {
		if v := countZeros(hll.registers); v != 0 {
			return uint64(linearCounting(m, v))
		}
	}

	return uint64(est)
}

func (hll *HyperLogLog) estimateBias(est float64) float64 {
	estTable, biasTable := rawEstimateData[hll.precision-4], biasData[hll.precision-4]

	if estTable[0] > est {
		return biasTable[0]
	}

	lastEstimate := estTable[len(estTable)-1]
	if lastEstimate < est {
		return biasTable[len(biasTable)-1]
	}

	var i int
	for i = 0; i < len(estTable) && estTable[i] < est; i++ {
	}

	e1, b1 := estTable[i-1], biasTable[i-1]
	e2, b2 := estTable[i], biasTable[i]

	c := (est - e1) / (e2 - e1)
	return b1*(1-c) + b2*c
}
