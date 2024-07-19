package negentropy

import (
	"errors"
	"sort"
)

type Vector struct {
	items []Item
}

func NewVector() *Vector {
	return &Vector{
		items: make([]Item, 0, 30),
	}
}

func (v *Vector) Insert(createdAt uint64, id string) error {
	// fmt.Fprintln(os.Stderr, "Insert", createdAt, id)
	if len(id) != IDSize*2 {
		return errors.New("bad id size for added item")
	}
	item := NewItem(createdAt, id)

	v.items = append(v.items, *item)
	return nil
}

func (v *Vector) Seal() error {
	sort.Slice(v.items, func(i, j int) bool {
		return v.items[i].LessThan(v.items[j])
	})

	for i := 1; i < len(v.items); i++ {
		if v.items[i-1].Equals(v.items[i]) {
			return errors.New("duplicate item inserted")
		}
	}
	return nil
}

func (v *Vector) Size() int {
	return len(v.items)
}

func (v *Vector) GetItem(i uint64) (Item, error) {
	if i >= uint64(len(v.items)) {
		return Item{}, errors.New("index out of bounds")
	}
	return v.items[i], nil
}

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
		out.AddItem(item)
		return true
	}); err != nil {
		return Fingerprint{}, err
	}

	return out.GetFingerprint(end - begin), nil
}
