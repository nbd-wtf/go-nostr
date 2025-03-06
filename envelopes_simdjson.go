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
			var filter Filter
			smp.TargetObject, smp.TargetInternalArray, err = filter.UnmarshalSIMD(
				&iter, smp.TargetObject, smp.TargetInternalArray)
			if err != nil {
				return nil, err
			}
			v.Filters = Filters{filter}
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
