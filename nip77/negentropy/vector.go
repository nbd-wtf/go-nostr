package negentropy

import (
	"encoding/hex"
	"fmt"
	"iter"
	"slices"

	"github.com/nbd-wtf/go-nostr"
)

type Vector struct {
	items  []Item
	sealed bool
}

func NewVector() *Vector {
	return &Vector{
		items: make([]Item, 0, 30),
	}
}

func (v *Vector) Insert(createdAt nostr.Timestamp, id string) error {
	if len(id)/2 != 32 {
		return fmt.Errorf("bad id size for added item: expected %d, got %d", 32, len(id)/2)
	}

	item := Item{createdAt, id}
	v.items = append(v.items, item)
	return nil
}

func (v *Vector) Size() int { return len(v.items) }

func (v *Vector) Seal() {
	if v.sealed {
		panic("trying to seal an already sealed vector")
	}
	v.sealed = true
	slices.SortFunc(v.items, itemCompare)
}

func (v *Vector) GetBound(idx int) Bound {
	if idx < len(v.items) {
		return Bound{v.items[idx]}
	}
	return infiniteBound
}

func (v *Vector) Range(begin, end int) iter.Seq2[int, Item] {
	return func(yield func(int, Item) bool) {
		for i := begin; i < end; i++ {
			if !yield(i, v.items[i]) {
				break
			}
		}
	}
}

func (v *Vector) FindLowerBound(begin, end int, bound Bound) int {
	idx, _ := slices.BinarySearchFunc(v.items[begin:end], bound.Item, itemCompare)
	return begin + idx
}

func (v *Vector) Fingerprint(begin, end int) [FingerprintSize]byte {
	var out Accumulator
	out.SetToZero()

	tmp := make([]byte, 32)
	for _, item := range v.Range(begin, end) {
		hex.Decode(tmp, []byte(item.ID))
		out.AddBytes(tmp)
	}

	return out.GetFingerprint(end - begin)
}
