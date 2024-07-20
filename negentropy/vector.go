package negentropy

import (
	"errors"
	"fmt"
	"sort"

	"github.com/nbd-wtf/go-nostr"
)

type Vector struct {
	items  []Item
	idSize int
}

func NewVector(idSize int) *Vector {
	return &Vector{
		items:  make([]Item, 0, 30),
		idSize: idSize,
	}
}

func (v *Vector) Insert(createdAt nostr.Timestamp, id string) error {
	// fmt.Fprintln(os.Stderr, "Insert", createdAt, id)
	if len(id)/2 != v.idSize {
		return fmt.Errorf("bad id size for added item: expected %d, got %d", v.idSize, len(id)/2)
	}

	item := Item{createdAt, id}
	v.items = append(v.items, item)
	return nil
}

func (v *Vector) Seal() error {
	sort.Slice(v.items, func(i, j int) bool {
		return v.items[i].LessThan(v.items[j])
	})

	for i := 1; i < len(v.items); i++ {
		if v.items[i-1].ID == v.items[i].ID {
			return errors.New("duplicate item inserted")
		}
	}
	return nil
}

func (v *Vector) Size() int   { return len(v.items) }
func (v *Vector) IDSize() int { return v.idSize }

func (v *Vector) Iterate(begin, end int, cb func(Item, int) bool) error {
	for i := begin; i < end; i++ {
		if !cb(v.items[i], i) {
			break
		}
	}
	return nil
}

func (v *Vector) FindLowerBound(begin, end int, bound Bound) (int, error) {
	i := sort.Search(len(v.items[begin:end]), func(i int) bool {
		return !v.items[begin+i].LessThan(bound.Item)
	})
	return begin + i, nil
}

func (v *Vector) Fingerprint(begin, end int) (Fingerprint, error) {
	var out Accumulator
	out.SetToZero()

	if err := v.Iterate(begin, end, func(item Item, _ int) bool {
		out.Add(item.ID)
		return true
	}); err != nil {
		return Fingerprint{}, err
	}

	return out.GetFingerprint(end - begin), nil
}
