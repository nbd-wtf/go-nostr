package vector

import (
	"encoding/hex"
	"fmt"
	"iter"
	"slices"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip77/negentropy"
	"github.com/nbd-wtf/go-nostr/nip77/negentropy/storage"
)

type Vector struct {
	items  []negentropy.Item
	sealed bool

	acc storage.Accumulator
}

func New() *Vector {
	return &Vector{
		items: make([]negentropy.Item, 0, 30),
	}
}

func (v *Vector) Insert(createdAt nostr.Timestamp, id string) {
	if len(id) != 64 {
		panic(fmt.Errorf("bad id size for added item: expected %d bytes, got %d", 32, len(id)/2))
	}

	item := negentropy.Item{Timestamp: createdAt, ID: id}
	v.items = append(v.items, item)
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

func (v *Vector) Fingerprint(begin, end int) string {
	v.acc.Reset()

	tmp := make([]byte, 32)
	for _, item := range v.Range(begin, end) {
		hex.Decode(tmp, []byte(item.ID))
		v.acc.AddBytes(tmp)
	}

	return v.acc.GetFingerprint(end - begin)
}
