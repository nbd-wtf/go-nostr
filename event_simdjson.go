package nostr

import (
	"bytes"
	"fmt"

	"github.com/minio/simdjson-go"
)

var (
	attrId        = []byte("id")
	attrPubkey    = []byte("pubkey")
	attrCreatedAt = []byte("created_at")
	attrKind      = []byte("kind")
	attrContent   = []byte("content")
	attrTags      = []byte("tags")
	attrSig       = []byte("sig")
)

func (event *Event) UnmarshalSIMD(
	iter *simdjson.Iter,
	obj *simdjson.Object,
	arr *simdjson.Array,
	subArr *simdjson.Array,
) (*simdjson.Object, *simdjson.Array, *simdjson.Array, error) {
	obj, err := iter.Object(obj)
	if err != nil {
		return obj, arr, subArr, fmt.Errorf("unexpected at event: %w", err)
	}

	for {
		name, t, err := obj.NextElementBytes(iter)
		if err != nil {
			return obj, arr, subArr, err
		} else if t == simdjson.TypeNone {
			break
		}

		switch {
		case bytes.Equal(name, attrId):
			event.ID, err = iter.String()
		case bytes.Equal(name, attrPubkey):
			event.PubKey, err = iter.String()
		case bytes.Equal(name, attrContent):
			event.Content, err = iter.String()
		case bytes.Equal(name, attrSig):
			event.Sig, err = iter.String()
		case bytes.Equal(name, attrCreatedAt):
			var ts uint64
			ts, err = iter.Uint()
			event.CreatedAt = Timestamp(ts)
		case bytes.Equal(name, attrKind):
			var kind uint64
			kind, err = iter.Uint()
			event.Kind = int(kind)
		case bytes.Equal(name, attrTags):
			arr, err = iter.Array(arr)
			if err != nil {
				return obj, arr, subArr, err
			}
			event.Tags = make(Tags, 0, 10)
			titer := arr.Iter()
			for {
				if t := titer.Advance(); t == simdjson.TypeNone {
					break
				}
				subArr, err = titer.Array(subArr)
				if err != nil {
					return obj, arr, subArr, err
				}
				tag, err := subArr.AsString()
				if err != nil {
					return obj, arr, subArr, err
				}
				event.Tags = append(event.Tags, tag)
			}
		default:
			return obj, arr, subArr, fmt.Errorf("unexpected event field '%s'", name)
		}

		if err != nil {
			return obj, arr, subArr, err
		}
	}

	return obj, arr, subArr, nil
}
