package negentropy

// Storage defines an interface for storage operations, similar to the abstract class in C++.
type Storage interface {
	Insert(uint64, string) error
	Seal() error

	Size() int
	GetItem(i uint64) (Item, error)
	Iterate(begin, end int, cb func(item Item, i int) bool) error
	FindLowerBound(begin, end int, value Bound) (int, error)
	Fingerprint(begin, end int) (Fingerprint, error)
}
