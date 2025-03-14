//go:build sonic

package nostr

import (
	"encoding/hex"
	stdlibjson "encoding/json"
	"fmt"
	"unsafe"

	"github.com/bytedance/sonic/ast"
)

type sonicVisitorPosition int

const (
	inEnvelope sonicVisitorPosition = iota

	inEvent
	inReq
	inOk
	inEose
	inCount
	inAuth
	inClose
	inClosed
	inNotice

	inFilterObject
	inEventObject
	inCountObject

	inSince
	inLimit
	inUntil
	inIds
	inAuthors
	inKinds
	inSearch
	inAFilterTag

	inId
	inCreatedAt
	inKind
	inContent
	inPubkey
	inSig
	inTags       // we just saw the "tags" object key
	inTagsList   // we have just seen the first `[` of the tags
	inAnEventTag // we are inside an actual tag, i.e we have just seen `[[`, or `].[`
)

func (spp sonicVisitorPosition) String() string {
	switch spp {
	case inEnvelope:
		return "inEnvelope"
	case inEvent:
		return "inEvent"
	case inReq:
		return "inReq"
	case inOk:
		return "inOk"
	case inEose:
		return "inEose"
	case inCount:
		return "inCount"
	case inAuth:
		return "inAuth"
	case inClose:
		return "inClose"
	case inClosed:
		return "inClosed"
	case inNotice:
		return "inNotice"
	case inFilterObject:
		return "inFilterObject"
	case inEventObject:
		return "inEventObject"
	case inCountObject:
		return "inCountObject"
	case inSince:
		return "inSince"
	case inLimit:
		return "inLimit"
	case inUntil:
		return "inUntil"
	case inIds:
		return "inIds"
	case inAuthors:
		return "inAuthors"
	case inKinds:
		return "inKinds"
	case inAFilterTag:
		return "inAFilterTag"
	case inId:
		return "inId"
	case inCreatedAt:
		return "inCreatedAt"
	case inKind:
		return "inKind"
	case inContent:
		return "inContent"
	case inPubkey:
		return "inPubkey"
	case inSig:
		return "inSig"
	case inTags:
		return "inTags"
	case inTagsList:
		return "inTagsList"
	case inAnEventTag:
		return "inAnEventTag"
	default:
		return "<unexpected-spp>"
	}
}

type sonicVisitor struct {
	event  *EventEnvelope
	req    *ReqEnvelope
	ok     *OKEnvelope
	eose   *EOSEEnvelope
	count  *CountEnvelope
	auth   *AuthEnvelope
	close  *CloseEnvelope
	closed *ClosedEnvelope
	notice *NoticeEnvelope

	whereWeAre sonicVisitorPosition

	currentEvent    *Event
	currentEventTag Tag

	currentFilter        *Filter
	currentFilterTagList []string
	currentFilterTagName string

	smp          *sonicMessageParser
	mainEnvelope Envelope
}

func (sv *sonicVisitor) OnArrayBegin(capacity int) error {
	// fmt.Println("***", "OnArrayBegin", "==", sv.whereWeAre)

	switch sv.whereWeAre {
	case inTags:
		sv.whereWeAre = inTagsList
		sv.currentEvent.Tags = sv.smp.reusableTagArray
	case inTagsList:
		sv.whereWeAre = inAnEventTag
		sv.currentEventTag = sv.smp.reusableStringArray
	case inAFilterTag:
		// we have already created this
	}

	return nil
}

func (sv *sonicVisitor) OnArrayEnd() error {
	// fmt.Println("***", "OnArrayEnd", "==", sv.whereWeAre)

	switch sv.whereWeAre {
	// envelopes
	case inEvent:
		sv.mainEnvelope = sv.event
	case inReq:
		sv.mainEnvelope = sv.req
		sv.smp.doneWithFilterSlice(sv.req.Filters)
	case inOk:
		sv.mainEnvelope = sv.ok
	case inEose:
		sv.mainEnvelope = sv.eose
	case inCount:
		sv.mainEnvelope = sv.count
	case inAuth:
		sv.mainEnvelope = sv.auth
	case inClose:
		sv.mainEnvelope = sv.close
	case inClosed:
		sv.mainEnvelope = sv.closed
	case inNotice:
		sv.mainEnvelope = sv.notice

		// filter object properties
	case inIds:
		sv.whereWeAre = inFilterObject
		sv.smp.doneWithStringSlice(sv.currentFilter.IDs)
	case inAuthors:
		sv.whereWeAre = inFilterObject
		sv.smp.doneWithStringSlice(sv.currentFilter.Authors)
	case inKinds:
		sv.whereWeAre = inFilterObject
		sv.smp.doneWithIntSlice(sv.currentFilter.Kinds)
	case inAFilterTag:
		sv.currentFilter.Tags[sv.currentFilterTagName] = sv.currentFilterTagList
		sv.whereWeAre = inFilterObject
		sv.smp.doneWithStringSlice(sv.currentFilterTagList)

		// event object properties
	case inAnEventTag:
		sv.currentEvent.Tags = append(sv.currentEvent.Tags, sv.currentEventTag)
		sv.whereWeAre = inTagsList
		sv.smp.doneWithStringSlice(sv.currentEventTag)
	case inTags, inTagsList:
		sv.whereWeAre = inEventObject
		sv.smp.doneWithTagSlice(sv.currentEvent.Tags)

	default:
		return fmt.Errorf("unexpected array end at %v", sv.whereWeAre)
	}
	return nil
}

func (sv *sonicVisitor) OnObjectBegin(capacity int) error {
	// fmt.Println("***", "OnObjectBegin", "==", sv.whereWeAre)

	switch sv.whereWeAre {
	case inEvent:
		sv.whereWeAre = inEventObject
		sv.currentEvent = &Event{}
	case inAuth:
		sv.whereWeAre = inEventObject
		sv.currentEvent = &Event{}
	case inReq:
		sv.whereWeAre = inFilterObject
		sv.currentFilter = &Filter{}
	case inCount:
		// set this temporarily, we will switch to a filterObject if we see "count" or "hll"
		sv.whereWeAre = inFilterObject
		sv.currentFilter = &Filter{}
	default:
		return fmt.Errorf("unexpected object begin at %v", sv.whereWeAre)
	}

	return nil
}

func (sv *sonicVisitor) OnObjectKey(key string) error {
	// fmt.Println("***", "OnObjectKey", key, "==", sv.whereWeAre)

	switch sv.whereWeAre {
	case inEventObject:
		switch key {
		case "id":
			sv.whereWeAre = inId
		case "sig":
			sv.whereWeAre = inSig
		case "pubkey":
			sv.whereWeAre = inPubkey
		case "content":
			sv.whereWeAre = inContent
		case "created_at":
			sv.whereWeAre = inCreatedAt
		case "kind":
			sv.whereWeAre = inKind
		case "tags":
			sv.whereWeAre = inTags
		default:
			return fmt.Errorf("unexpected event attr %s", key)
		}
	case inFilterObject:
		switch key {
		case "limit":
			sv.whereWeAre = inLimit
		case "since":
			sv.whereWeAre = inSince
		case "until":
			sv.whereWeAre = inUntil
		case "ids":
			sv.whereWeAre = inIds
			sv.currentFilter.IDs = sv.smp.reusableStringArray
		case "authors":
			sv.whereWeAre = inAuthors
			sv.currentFilter.Authors = sv.smp.reusableStringArray
		case "kinds":
			sv.whereWeAre = inKinds
			sv.currentFilter.Kinds = sv.smp.reusableIntArray
		case "search":
			sv.whereWeAre = inSearch
		case "count", "hll":
			// oops, switch to a countObject
			sv.whereWeAre = inCountObject
		default:
			if len(key) > 1 && key[0] == '#' {
				if sv.currentFilter.Tags == nil {
					sv.currentFilter.Tags = make(TagMap, 1)
				}
				sv.currentFilterTagList = sv.smp.reusableStringArray
				sv.currentFilterTagName = key[1:]
				sv.whereWeAre = inAFilterTag
			} else {
				return fmt.Errorf("unexpected filter attr %s", key)
			}
		}
	case inCountObject:
		// we'll judge by the shape of the value so ignore this
	default:
		return fmt.Errorf("unexpected object key %s at %s", key, sv.whereWeAre)
	}

	return nil
}

func (sv *sonicVisitor) OnObjectEnd() error {
	// fmt.Println("***", "OnObjectEnd", "==", sv.whereWeAre)

	switch sv.whereWeAre {
	case inEventObject:
		if sv.event != nil {
			sv.event.Event = *sv.currentEvent
			sv.whereWeAre = inEvent
		} else {
			sv.auth.Event = *sv.currentEvent
			sv.whereWeAre = inAuth
		}
		sv.currentEvent = nil
	case inFilterObject:
		if sv.req != nil {
			sv.req.Filters = append(sv.req.Filters, *sv.currentFilter)
			sv.whereWeAre = inReq
		} else {
			sv.count.Filter = *sv.currentFilter
			sv.whereWeAre = inCount
		}
		sv.currentFilter = nil
	case inCountObject:
		sv.whereWeAre = inCount
	default:
		return fmt.Errorf("unexpected object end at %s", sv.whereWeAre)
	}

	return nil
}

func (sv *sonicVisitor) OnString(v string) error {
	// fmt.Println("***", "OnString", v, "==", sv.whereWeAre)

	switch sv.whereWeAre {
	case inEnvelope:
		switch v {
		case "EVENT":
			sv.event = &EventEnvelope{}
			sv.whereWeAre = inEvent
		case "REQ":
			sv.req = &ReqEnvelope{Filters: sv.smp.reusableFilterArray}
			sv.whereWeAre = inReq
		case "OK":
			sv.ok = &OKEnvelope{}
			sv.whereWeAre = inOk
		case "EOSE":
			sv.whereWeAre = inEose
		case "COUNT":
			sv.count = &CountEnvelope{}
			sv.whereWeAre = inCount
		case "AUTH":
			sv.auth = &AuthEnvelope{}
			sv.whereWeAre = inAuth
		case "CLOSE":
			sv.whereWeAre = inClose
		case "CLOSED":
			sv.closed = &ClosedEnvelope{}
			sv.whereWeAre = inClosed
		case "NOTICE":
			sv.whereWeAre = inNotice
		default:
			return UnknownLabel
		}

		// in an envelope
	case inEvent:
		sv.event.SubscriptionID = &v
	case inReq:
		sv.req.SubscriptionID = v
	case inOk:
		if sv.ok.EventID == "" {
			sv.ok.EventID = v
		} else {
			sv.ok.Reason = v
		}
	case inEose:
		sv.eose = (*EOSEEnvelope)(&v)
	case inCount:
		sv.count.SubscriptionID = v
	case inAuth:
		sv.auth.Challenge = &v
	case inClose:
		sv.close = (*CloseEnvelope)(&v)
	case inClosed:
		if sv.closed.SubscriptionID == "" {
			sv.closed.SubscriptionID = v
		} else {
			sv.closed.Reason = v
		}
	case inNotice:
		sv.notice = (*NoticeEnvelope)(&v)

		// filter object properties
	case inIds:
		sv.currentFilter.IDs = append(sv.currentFilter.IDs, v)
	case inAuthors:
		sv.currentFilter.Authors = append(sv.currentFilter.Authors, v)
	case inSearch:
		sv.currentFilter.Search = v
		sv.whereWeAre = inFilterObject
	case inAFilterTag:
		sv.currentFilterTagList = append(sv.currentFilterTagList, v)

		// id object properties
	case inId:
		sv.currentEvent.ID = v
		sv.whereWeAre = inEventObject
	case inContent:
		sv.currentEvent.Content = v
		sv.whereWeAre = inEventObject
	case inPubkey:
		sv.currentEvent.PubKey = v
		sv.whereWeAre = inEventObject
	case inSig:
		sv.currentEvent.Sig = v
		sv.whereWeAre = inEventObject
	case inAnEventTag:
		sv.currentEventTag = append(sv.currentEventTag, v)

		// count object properties
	case inCountObject:
		sv.count.HyperLogLog, _ = hex.DecodeString(v)

	default:
		return fmt.Errorf("unexpected string %s at %v", v, sv.whereWeAre)
	}
	return nil
}

func (sv *sonicVisitor) OnInt64(v int64, _ stdlibjson.Number) error {
	// fmt.Println("***", "OnInt64", v, "==", sv.whereWeAre)

	switch sv.whereWeAre {
	// event object
	case inCreatedAt:
		sv.currentEvent.CreatedAt = Timestamp(v)
		sv.whereWeAre = inEventObject
	case inKind:
		sv.currentEvent.Kind = int(v)
		sv.whereWeAre = inEventObject

	// filter object
	case inLimit:
		sv.currentFilter.Limit = int(v)
		sv.currentFilter.LimitZero = v == 0
		sv.whereWeAre = inFilterObject
	case inSince:
		sv.currentFilter.Since = (*Timestamp)(&v)
		sv.whereWeAre = inFilterObject
	case inUntil:
		sv.currentFilter.Until = (*Timestamp)(&v)
		sv.whereWeAre = inFilterObject
	case inKinds:
		sv.currentFilter.Kinds = append(sv.currentFilter.Kinds, int(v))

	// count object
	case inCountObject:
		sv.count.Count = &v
	}
	return nil
}

func (sv *sonicVisitor) OnBool(v bool) error {
	// fmt.Println("***", "OnBool", v, "==", sv.whereWeAre)

	if sv.whereWeAre == inOk {
		sv.ok.OK = v
		return nil
	} else {
		return fmt.Errorf("unexpected boolean")
	}
}

func (_ sonicVisitor) OnNull() error {
	return fmt.Errorf("null shouldn't be anywhere in a message")
}

func (_ sonicVisitor) OnFloat64(v float64, n stdlibjson.Number) error {
	return fmt.Errorf("float shouldn't be anywhere in a message")
}

type sonicMessageParser struct {
	reusableFilterArray []Filter
	reusableTagArray    []Tag
	reusableStringArray []string
	reusableIntArray    []int
}

// NewMessageParser returns a sonicMessageParser object that is intended to be reused many times.
// It is not goroutine-safe.
func NewMessageParser() sonicMessageParser {
	return sonicMessageParser{
		reusableFilterArray: make([]Filter, 0, 1000),
		reusableTagArray:    make([]Tag, 0, 10000),
		reusableStringArray: make([]string, 0, 10000),
		reusableIntArray:    make([]int, 0, 10000),
	}
}

var NewSonicMessageParser = NewMessageParser

func (smp *sonicMessageParser) doneWithFilterSlice(slice []Filter) {
	if unsafe.SliceData(smp.reusableFilterArray) == unsafe.SliceData(slice) {
		smp.reusableFilterArray = slice[len(slice):]
	}

	if cap(smp.reusableFilterArray) < 7 {
		// create a new one
		smp.reusableFilterArray = make([]Filter, 0, 1000)
	}
}

func (smp *sonicMessageParser) doneWithTagSlice(slice []Tag) {
	if unsafe.SliceData(smp.reusableTagArray) == unsafe.SliceData(slice) {
		smp.reusableTagArray = slice[len(slice):]
	}

	if cap(smp.reusableTagArray) < 7 {
		// create a new one
		smp.reusableTagArray = make([]Tag, 0, 10000)
	}
}

func (smp *sonicMessageParser) doneWithStringSlice(slice []string) {
	if unsafe.SliceData(smp.reusableStringArray) == unsafe.SliceData(slice) {
		smp.reusableStringArray = slice[len(slice):]
	}

	if cap(smp.reusableStringArray) < 15 {
		// create a new one
		smp.reusableStringArray = make([]string, 0, 10000)
	}
}

func (smp *sonicMessageParser) doneWithIntSlice(slice []int) {
	if unsafe.SliceData(smp.reusableIntArray) == unsafe.SliceData(slice) {
		smp.reusableIntArray = slice[len(slice):]
	}

	if cap(smp.reusableIntArray) < 8 {
		// create a new one
		smp.reusableIntArray = make([]int, 0, 10000)
	}
}

// ParseMessage parses a message like ["EVENT", ...] or ["REQ", ...] and returns an Envelope.
// The returned envelopes, filters and events' slices should not be appended to, otherwise stuff
// will break.
//
// When an unexpected message (like ["NEG-OPEN", ...]) is found, the error UnknownLabel will be
// returned. Other errors will be returned if the JSON is malformed or the objects are not exactly
// as they should.
func (smp sonicMessageParser) ParseMessage(message string) (Envelope, error) {
	sv := &sonicVisitor{smp: &smp}
	sv.whereWeAre = inEnvelope

	err := ast.Preorder(message, sv, nil)

	return sv.mainEnvelope, err
}
