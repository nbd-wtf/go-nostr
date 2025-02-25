package nostr

import (
	"fmt"
	"slices"

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

func (event *Event) UnmarshalSIMD(iter *simdjson.Iter) error {
	obj, err := iter.Object(nil)
	if err != nil {
		return fmt.Errorf("unexpected at event: %w", err)
	}

	for {
		name, t, err := obj.NextElementBytes(iter)
		if err != nil {
			return err
		} else if t == simdjson.TypeNone {
			break
		}

		switch {
		case slices.Equal(name, attrId):
			event.ID, err = iter.String()
		case slices.Equal(name, attrPubkey):
			event.PubKey, err = iter.String()
		case slices.Equal(name, attrContent):
			event.Content, err = iter.String()
		case slices.Equal(name, attrSig):
			event.Sig, err = iter.String()
		case slices.Equal(name, attrCreatedAt):
			var ts uint64
			ts, err = iter.Uint()
			event.CreatedAt = Timestamp(ts)
		case slices.Equal(name, attrKind):
			var kind uint64
			kind, err = iter.Uint()
			event.Kind = int(kind)
		case slices.Equal(name, attrTags):
			var arr *simdjson.Array
			arr, err = iter.Array(nil)
			if err != nil {
				return err
			}
			event.Tags = make(Tags, 0, 10)
			titer := arr.Iter()
			var subArr *simdjson.Array
			for {
				if t := titer.Advance(); t == simdjson.TypeNone {
					break
				}
				subArr, err = titer.Array(subArr)
				if err != nil {
					return err
				}
				tag, err := subArr.AsString()
				if err != nil {
					return err
				}
				event.Tags = append(event.Tags, tag)
			}
		default:
			return fmt.Errorf("unexpected event field '%s'", name)
		}

		if err != nil {
			return err
		}
	}

	return nil
}
