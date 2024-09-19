package vector

import (
	"encoding/hex"
	"fmt"
	"iter"
	"slices"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip77/negentropy"
)

type Vector struct {
	items  []negentropy.Item
	sealed bool
}

func New() *Vector {
	return &Vector{
		items: make([]negentropy.Item, 0, 30),
	}
}

func (v *Vector) Insert(createdAt nostr.Timestamp, id string) error {
	if len(id) != 64 {
		return fmt.Errorf("bad id size for added item: expected %d bytes, got %d", 32, len(id)/2)
	}

	item := negentropy.Item{Timestamp: createdAt, ID: id}
	v.items = append(v.items, item)
	return nil
}

func (v *Vector) Size() int { return len(v.items) }

func (v *Vector) Seal() {
	if v.sealed {
		panic("trying to seal an already sealed vector")
	}
	v.sealed = true
	slices.SortFunc(v.items, negentropy.ItemCompare)
}

func (v *Vector) GetBound(idx int) negentropy.Bound {
	if idx < len(v.items) {
		return negentropy.Bound{Item: v.items[idx]}
	}
	return negentropy.InfiniteBound
}

func (v *Vector) Range(begin, end int) iter.Seq2[int, negentropy.Item] {
	return func(yield func(int, negentropy.Item) bool) {
		for i := begin; i < end; i++ {
			if !yield(i, v.items[i]) {
				break
			}
		}
	}
}

func (v *Vector) FindLowerBound(begin, end int, bound negentropy.Bound) int {
	idx, _ := slices.BinarySearchFunc(v.items[begin:end], bound.Item, negentropy.ItemCompare)
	return begin + idx
}

func (v *Vector) Fingerprint(begin, end int) [negentropy.FingerprintSize]byte {
	var out negentropy.Accumulator
	out.SetToZero()

	tmp := make([]byte, 32)
	for _, item := range v.Range(begin, end) {
		hex.Decode(tmp, []byte(item.ID))
		out.AddBytes(tmp)
	}

	return out.GetFingerprint(end - begin)
}
