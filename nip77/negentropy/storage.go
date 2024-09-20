package negentropy

import (
	"iter"
)

type Storage interface {
	Size() int
	Range(begin, end int) iter.Seq2[int, Item]
	FindLowerBound(begin, end int, value Bound) int
	GetBound(idx int) Bound
	Fingerprint(begin, end int) string
}
