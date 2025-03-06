package nostr

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/minio/simdjson-go"
)

type SIMDMessageParser struct {
	ParsedJSON          *simdjson.ParsedJson
	TopLevelArray       *simdjson.Array  // used for the top-level envelope
	TargetObject        *simdjson.Object // used for the event object itself, or for the count object, or the filter object
	TargetInternalArray *simdjson.Array  // used for tags array inside the event or each of the values in a filter
	AuxArray            *simdjson.Array  // used either for each of the tags inside the event or for each of the multiple filters that may code
	AuxIter             *simdjson.Iter
}

func (smp *SIMDMessageParser) ParseMessage(message []byte) (Envelope, error) {
	var err error

	smp.ParsedJSON, err = simdjson.Parse(message, smp.ParsedJSON)
	if err != nil {
		return nil, fmt.Errorf("simdjson parse failed: %w", err)
	}

	iter := smp.ParsedJSON.Iter()
	iter.AdvanceInto()
	if t := iter.Advance(); t != simdjson.TypeArray {
		return nil, fmt.Errorf("top-level must be an array")
	}
	arr, _ := iter.Array(nil)
	iter = arr.Iter()
	iter.Advance()
	label, _ := iter.StringBytes()

	switch {
	case bytes.Equal(label, labelEvent):
		v := &EventEnvelope{}
		// we may or may not have a subscription ID, so peek
		if iter.PeekNext() == simdjson.TypeString {
			iter.Advance()
			// we have a subscription ID
			subID, err := iter.String()
			if err != nil {
				return nil, err
			}
			v.SubscriptionID = &subID
		}
		// now get the event
		if typ := iter.Advance(); typ == simdjson.TypeNone {
			return nil, fmt.Errorf("missing event")
		}

		smp.TargetObject, smp.TargetInternalArray, smp.AuxArray, err = v.Event.UnmarshalSIMD(
			&iter, smp.TargetObject, smp.TargetInternalArray, smp.AuxArray)
		return v, err
	case bytes.Equal(label, labelReq):
		v := &ReqEnvelope{}

		// we must have a subscription id
		if typ := iter.Advance(); typ == simdjson.TypeString {
			v.SubscriptionID, err = iter.String()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unexpected %s for REQ subscription id", typ)
		}

		// now get the filters
		v.Filters = make(Filters, 0, 1)
		for {
			if typ, err := iter.AdvanceIter(smp.AuxIter); err != nil {
				return nil, err
			} else if typ == simdjson.TypeNone {
				break
			} else {
			}

			var filter Filter
			smp.TargetObject, smp.TargetInternalArray, err = filter.UnmarshalSIMD(
				smp.AuxIter, smp.TargetObject, smp.TargetInternalArray)
			if err != nil {
				return nil, err
			}
			v.Filters = append(v.Filters, filter)
		}

		if len(v.Filters) == 0 {
			return nil, fmt.Errorf("need at least one filter")
		}

		return v, nil
	case bytes.Equal(label, labelCount):
		v := &CountEnvelope{}
		// this has two cases:
		// in the first case (request from client) this is like REQ except with always one filter
		// in the other (response from relay) we have a json object response
		// but both cases start with a subscription id

		if typ := iter.Advance(); typ == simdjson.TypeString {
			v.SubscriptionID, err = iter.String()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unexpected %s for COUNT subscription id", typ)
		}

		// now get either a single filter or stuff from the json object
		if typ := iter.Advance(); typ == simdjson.TypeNone {
			return nil, fmt.Errorf("missing json object")
		}

		if el, err := iter.FindElement(nil, "count"); err == nil {
			c, _ := el.Iter.Uint()
			count := int64(c)
			v.Count = &count
			if el, err = iter.FindElement(nil, "hll"); err == nil {
				if hllHex, err := el.Iter.StringBytes(); err != nil || len(hllHex) != 512 {
					return nil, fmt.Errorf("hll is malformed")
				} else {
					v.HyperLogLog = make([]byte, 256)
					if _, err := hex.Decode(v.HyperLogLog, hllHex); err != nil {
						return nil, fmt.Errorf("hll is invalid hex")
					}
				}
			}
		} else {
			smp.TargetObject, smp.TargetInternalArray, err = v.Filter.UnmarshalSIMD(
				&iter, smp.TargetObject, smp.TargetInternalArray)
			if err != nil {
				return nil, err
			}
		}

		return v, nil
	case bytes.Equal(label, labelNotice):
		x := NoticeEnvelope("")
		v := &x
		if typ := iter.Advance(); typ == simdjson.TypeString {
			msg, _ := iter.String()
			*v = NoticeEnvelope(msg)
		}
		return v, nil
	case bytes.Equal(label, labelEose):
		x := EOSEEnvelope("")
		v := &x
		if typ := iter.Advance(); typ == simdjson.TypeString {
			msg, _ := iter.String()
			*v = EOSEEnvelope(msg)
		}
		return v, nil
	case bytes.Equal(label, labelOk):
		v := &OKEnvelope{}
		if typ := iter.Advance(); typ == simdjson.TypeString {
			v.EventID, _ = iter.String()
		} else {
			return nil, fmt.Errorf("unexpected %s for OK id", typ)
		}
		if typ := iter.Advance(); typ == simdjson.TypeBool {
			v.OK, _ = iter.Bool()
		} else {
			return nil, fmt.Errorf("unexpected %s for OK status", typ)
		}
		if typ := iter.Advance(); typ == simdjson.TypeString {
			v.Reason, _ = iter.String()
		}
		return v, nil
	case bytes.Equal(label, labelAuth):
		v := &AuthEnvelope{}
		if typ := iter.Advance(); typ == simdjson.TypeString {
			// we have a challenge
			subID, err := iter.String()
			if err != nil {
				return nil, err
			}
			v.Challenge = &subID
			return v, nil
		} else {
			// we have an event
			smp.TargetObject, smp.TargetInternalArray, smp.AuxArray, err = v.Event.UnmarshalSIMD(
				&iter, smp.TargetObject, smp.TargetInternalArray, smp.AuxArray)
			return v, err
		}
	case bytes.Equal(label, labelClosed):
		v := &ClosedEnvelope{}
		if typ := iter.Advance(); typ == simdjson.TypeString {
			v.SubscriptionID, _ = iter.String()
		}
		if typ := iter.Advance(); typ == simdjson.TypeString {
			v.Reason, _ = iter.String()
		}
		return v, nil
	case bytes.Equal(label, labelClose):
		x := CloseEnvelope("")
		v := &x
		if typ := iter.Advance(); typ == simdjson.TypeString {
			msg, _ := iter.String()
			*v = CloseEnvelope(msg)
		}
		return v, nil
	default:
		return nil, UnknownLabel
	}
}

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
