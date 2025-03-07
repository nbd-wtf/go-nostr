package nostr

import (
	"encoding/hex"
	stdlibjson "encoding/json"
	"fmt"

	"github.com/bytedance/sonic/ast"
)

type sonicParserPosition int

const (
	inEnvelope sonicParserPosition = iota

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
	inTags
	inAnEventTag
)

func (spp sonicParserPosition) String() string {
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
	case inAnEventTag:
		return "inAnEventTag"
	default:
		return "<unexpected-spp>"
	}
}

type SonicMessageParser struct {
	event  *EventEnvelope
	req    *ReqEnvelope
	ok     *OKEnvelope
	eose   *EOSEEnvelope
	count  *CountEnvelope
	auth   *AuthEnvelope
	close  *CloseEnvelope
	closed *ClosedEnvelope
	notice *NoticeEnvelope

	whereWeAre sonicParserPosition

	currentEvent    *Event
	currentEventTag Tag

	currentFilter        *Filter
	currentFilterTagList []string
	currentFilterTagName string

	mainEnvelope Envelope
}

func (smp *SonicMessageParser) OnArrayBegin(capacity int) error {
	// fmt.Println("***", "OnArrayBegin", "==", smp.whereWeAre)

	switch smp.whereWeAre {
	case inTags:
		if smp.currentEvent.Tags == nil {
			smp.currentEvent.Tags = make(Tags, 0, 10)
			smp.currentEventTag = make(Tag, 0, 20)
		} else {
			smp.whereWeAre = inAnEventTag
		}
	case inAFilterTag:
		// we have already created this
	}

	return nil
}

func (smp *SonicMessageParser) OnArrayEnd() error {
	// fmt.Println("***", "OnArrayEnd", "==", smp.whereWeAre)

	switch smp.whereWeAre {
	// envelopes
	case inEvent:
		smp.mainEnvelope = smp.event
	case inReq:
		smp.mainEnvelope = smp.req
	case inOk:
		smp.mainEnvelope = smp.ok
	case inEose:
		smp.mainEnvelope = smp.eose
	case inCount:
		smp.mainEnvelope = smp.count
	case inAuth:
		smp.mainEnvelope = smp.auth
	case inClose:
		smp.mainEnvelope = smp.close
	case inClosed:
		smp.mainEnvelope = smp.closed
	case inNotice:
		smp.mainEnvelope = smp.notice

		// filter object properties
	case inIds, inAuthors, inKinds, inSearch:
		smp.whereWeAre = inFilterObject
	case inAFilterTag:
		smp.currentFilter.Tags[smp.currentFilterTagName] = smp.currentFilterTagList
		// reuse the same underlying slice because we know nothing else will be appended to it
		smp.currentFilterTagList = smp.currentFilterTagList[len(smp.currentFilterTagList):]
		smp.whereWeAre = inFilterObject

		// event object properties
	case inAnEventTag:
		smp.currentEvent.Tags = append(smp.currentEvent.Tags, smp.currentEventTag)
		// reuse the same underlying slice because we know nothing else will be appended to it
		smp.currentEventTag = smp.currentEventTag[len(smp.currentEventTag):]
		smp.whereWeAre = inTags
	case inTags:
		smp.whereWeAre = inEventObject

	default:
		return fmt.Errorf("unexpected array end at %v", smp.whereWeAre)
	}
	return nil
}

func (smp *SonicMessageParser) OnObjectBegin(capacity int) error {
	// fmt.Println("***", "OnObjectBegin", "==", smp.whereWeAre)

	switch smp.whereWeAre {
	case inEvent:
		smp.whereWeAre = inEventObject
		smp.currentEvent = &Event{}
	case inAuth:
		smp.whereWeAre = inEventObject
		smp.currentEvent = &Event{}
	case inReq:
		smp.whereWeAre = inFilterObject
		smp.currentFilter = &Filter{}
	case inCount:
		// set this temporarily, we will switch to a filterObject if we see "count" or "hll"
		smp.whereWeAre = inFilterObject
		smp.currentFilter = &Filter{}
	default:
		return fmt.Errorf("unexpected object begin at %v", smp.whereWeAre)
	}

	return nil
}

func (smp *SonicMessageParser) OnObjectKey(key string) error {
	// fmt.Println("***", "OnObjectKey", key, "==", smp.whereWeAre)

	switch smp.whereWeAre {
	case inEventObject:
		switch key {
		case "id":
			smp.whereWeAre = inId
		case "sig":
			smp.whereWeAre = inSig
		case "pubkey":
			smp.whereWeAre = inPubkey
		case "content":
			smp.whereWeAre = inContent
		case "created_at":
			smp.whereWeAre = inCreatedAt
		case "kind":
			smp.whereWeAre = inKind
		case "tags":
			smp.whereWeAre = inTags
		default:
			return fmt.Errorf("unexpected event attr %s", key)
		}
	case inFilterObject:
		switch key {
		case "limit":
			smp.whereWeAre = inLimit
		case "since":
			smp.whereWeAre = inSince
		case "until":
			smp.whereWeAre = inUntil
		case "ids":
			smp.whereWeAre = inIds
			smp.currentFilter.IDs = make([]string, 0, 25)
		case "authors":
			smp.whereWeAre = inAuthors
			smp.currentFilter.Authors = make([]string, 0, 25)
		case "kinds":
			smp.whereWeAre = inKinds
			smp.currentFilter.IDs = make([]string, 0, 12)
		case "search":
			smp.whereWeAre = inSearch
		case "count", "hll":
			// oops, switch to a countObject
			smp.whereWeAre = inCountObject
		default:
			if len(key) > 1 && key[0] == '#' {
				if smp.currentFilter.Tags == nil {
					smp.currentFilter.Tags = make(TagMap, 1)
					smp.currentFilterTagList = make([]string, 0, 25)
				}
				smp.whereWeAre = inAFilterTag
				smp.currentFilterTagName = key[1:]
			} else {
				return fmt.Errorf("unexpected filter attr %s", key)
			}
		}
	case inCountObject:
		// we'll judge by the shape of the value so ignore this
	default:
		return fmt.Errorf("unexpected object key %s at %s", key, smp.whereWeAre)
	}

	return nil
}

func (smp *SonicMessageParser) OnObjectEnd() error {
	// fmt.Println("***", "OnObjectEnd", "==", smp.whereWeAre)

	switch smp.whereWeAre {
	case inEventObject:
		if smp.event != nil {
			smp.event.Event = *smp.currentEvent
			smp.whereWeAre = inEvent
		} else {
			smp.auth.Event = *smp.currentEvent
			smp.whereWeAre = inAuth
		}
	case inFilterObject:
		if smp.req != nil {
			smp.req.Filters = append(smp.req.Filters, *smp.currentFilter)
			smp.whereWeAre = inReq
		} else {
			smp.count.Filter = *smp.currentFilter
			smp.whereWeAre = inCount
		}
	case inCountObject:
		smp.whereWeAre = inCount
	default:
		return fmt.Errorf("unexpected object end at %s", smp.whereWeAre)
	}

	return nil
}

func (smp *SonicMessageParser) OnString(v string) error {
	// fmt.Println("***", "OnString", v, "==", smp.whereWeAre)

	switch smp.whereWeAre {
	case inEnvelope:
		switch v {
		case "EVENT":
			smp.event = &EventEnvelope{}
			smp.whereWeAre = inEvent
		case "REQ":
			smp.req = &ReqEnvelope{Filters: make(Filters, 0, 1)}
			smp.whereWeAre = inReq
		case "OK":
			smp.ok = &OKEnvelope{}
			smp.whereWeAre = inOk
		case "EOSE":
			smp.whereWeAre = inEose
		case "COUNT":
			smp.count = &CountEnvelope{}
			smp.whereWeAre = inCount
		case "AUTH":
			smp.auth = &AuthEnvelope{}
			smp.whereWeAre = inAuth
		case "CLOSE":
			smp.whereWeAre = inClose
		case "CLOSED":
			smp.closed = &ClosedEnvelope{}
			smp.whereWeAre = inClosed
		case "NOTICE":
			smp.whereWeAre = inNotice
		}

		// in an envelope
	case inEvent:
		smp.event.SubscriptionID = &v
	case inReq:
		smp.req.SubscriptionID = v
	case inOk:
		if smp.ok.EventID == "" {
			smp.ok.EventID = v
		} else {
			smp.ok.Reason = v
		}
	case inEose:
		smp.eose = (*EOSEEnvelope)(&v)
	case inCount:
		smp.count.SubscriptionID = v
	case inAuth:
		smp.auth.Challenge = &v
	case inClose:
		smp.close = (*CloseEnvelope)(&v)
	case inClosed:
		if smp.closed.SubscriptionID == "" {
			smp.closed.SubscriptionID = v
		} else {
			smp.closed.Reason = v
		}
	case inNotice:
		smp.notice = (*NoticeEnvelope)(&v)

		// filter object properties
	case inIds:
		smp.currentFilter.IDs = append(smp.currentFilter.IDs, v)
	case inAuthors:
		smp.currentFilter.Authors = append(smp.currentFilter.Authors, v)
	case inSearch:
		smp.currentFilter.Search = v
		smp.whereWeAre = inFilterObject
	case inAFilterTag:
		smp.currentFilterTagList = append(smp.currentFilterTagList, v)

		// id object properties
	case inId:
		smp.currentEvent.ID = v
		smp.whereWeAre = inEventObject
	case inContent:
		smp.currentEvent.Content = v
		smp.whereWeAre = inEventObject
	case inPubkey:
		smp.currentEvent.PubKey = v
		smp.whereWeAre = inEventObject
	case inSig:
		smp.currentEvent.Sig = v
		smp.whereWeAre = inEventObject
	case inAnEventTag:
		smp.currentEventTag = append(smp.currentEventTag, v)

		// count object properties
	case inCountObject:
		smp.count.HyperLogLog, _ = hex.DecodeString(v)

	default:
		return fmt.Errorf("unexpected string %s at %v", v, smp.whereWeAre)
	}
	return nil
}

func (smp *SonicMessageParser) OnInt64(v int64, _ stdlibjson.Number) error {
	// fmt.Println("***", "OnInt64", v, "==", smp.whereWeAre)

	switch smp.whereWeAre {
	// event object
	case inCreatedAt:
		smp.currentEvent.CreatedAt = Timestamp(v)
		smp.whereWeAre = inEventObject
	case inKind:
		smp.currentEvent.Kind = int(v)
		smp.whereWeAre = inEventObject

	// filter object
	case inLimit:
		smp.currentFilter.Limit = int(v)
		smp.currentFilter.LimitZero = v == 0
	case inSince:
		smp.currentFilter.Since = (*Timestamp)(&v)
	case inUntil:
		smp.currentFilter.Until = (*Timestamp)(&v)
	case inKinds:
		smp.currentFilter.Kinds = append(smp.currentFilter.Kinds, int(v))

	// count object
	case inCountObject:
		smp.count.Count = &v
	}
	return nil
}

func (smp *SonicMessageParser) OnBool(v bool) error {
	// fmt.Println("***", "OnBool", v, "==", smp.whereWeAre)

	if smp.whereWeAre == inOk {
		smp.ok.OK = v
		return nil
	} else {
		return fmt.Errorf("unexpected boolean")
	}
}

func (_ SonicMessageParser) OnNull() error {
	return fmt.Errorf("null shouldn't be anywhere in a message")
}

func (_ SonicMessageParser) OnFloat64(v float64, n stdlibjson.Number) error {
	return fmt.Errorf("float shouldn't be anywhere in a message")
}

func ParseMessageSonic(message []byte) (Envelope, error) {
	smp := &SonicMessageParser{}
	smp.whereWeAre = inEnvelope

	err := ast.Preorder(string(message), smp, nil)

	return smp.mainEnvelope, err
}
