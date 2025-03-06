package nostr

import (
	"bytes"
	"fmt"

	"github.com/minio/simdjson-go"
)

var (
	attrIds     = []byte("ids")
	attrAuthors = []byte("authors")
	attrKinds   = []byte("kinds")
	attrLimit   = []byte("limit")
	attrSince   = []byte("since")
	attrUntil   = []byte("until")
	attrSearch  = []byte("search")
)

func (filter *Filter) UnmarshalSIMD(
	iter *simdjson.Iter,
	obj *simdjson.Object,
	arr *simdjson.Array,
) (*simdjson.Object, *simdjson.Array, error) {
	obj, err := iter.Object(obj)
	if err != nil {
		return obj, arr, fmt.Errorf("unexpected at filter: %w", err)
	}

	for {
		name, t, err := obj.NextElementBytes(iter)
		if err != nil {
			return obj, arr, err
		} else if t == simdjson.TypeNone {
			break
		}

		switch {
		case bytes.Equal(name, attrIds):
			if arr, err = iter.Array(arr); err == nil {
				filter.IDs, err = arr.AsString()
			}
		case bytes.Equal(name, attrAuthors):
			if arr, err = iter.Array(arr); err == nil {
				filter.Authors, err = arr.AsString()
			}
		case bytes.Equal(name, attrKinds):
			if arr, err = iter.Array(arr); err == nil {
				i := arr.Iter()
				filter.Kinds = make([]int, 0, 6)
				for {
					t := i.Advance()
					if t == simdjson.TypeNone {
						break
					}
					if kind, err := i.Uint(); err != nil {
						return obj, arr, err
					} else {
						filter.Kinds = append(filter.Kinds, int(kind))
					}
				}
			}
		case bytes.Equal(name, attrSearch):
			filter.Search, err = iter.String()
		case bytes.Equal(name, attrSince):
			var tsu uint64
			tsu, err = iter.Uint()
			ts := Timestamp(tsu)
			filter.Since = &ts
		case bytes.Equal(name, attrUntil):
			var tsu uint64
			tsu, err = iter.Uint()
			ts := Timestamp(tsu)
			filter.Until = &ts
		case bytes.Equal(name, attrLimit):
			var limit uint64
			limit, err = iter.Uint()
			filter.Limit = int(limit)
			if limit == 0 {
				filter.LimitZero = true
			}
		default:
			if len(name) > 1 && name[0] == '#' {
				if filter.Tags == nil {
					filter.Tags = make(TagMap, 1)
				}

				arr, err := iter.Array(arr)
				if err != nil {
					return obj, arr, err
				}
				vals, err := arr.AsString()
				if err != nil {
					return obj, arr, err
				}

				filter.Tags[string(name[1:])] = vals
				continue
			}

			return obj, arr, fmt.Errorf("unexpected filter field '%s'", name)
		}

		if err != nil {
			return obj, arr, err
		}
	}

	return obj, arr, nil
}
