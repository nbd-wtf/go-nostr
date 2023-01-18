package nostr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/valyala/fastjson"
	"time"
)

func (evt *Event) UnmarshalJSON(payload []byte) error {
	var fastjsonParser fastjson.Parser
	parsed, err := fastjsonParser.ParseBytes(payload)
	if err != nil {
		return fmt.Errorf("failed to parse event: %w", err)
	}

	obj, err := parsed.Object()
	if err != nil {
		return fmt.Errorf("event is not an object")
	}

	// prepare this to receive any extra property that may serialized along with the event
	evt.extra = make(map[string]any)

	var visiterr error
	obj.Visit(func(k []byte, v *fastjson.Value) {
		key := string(k)
		switch key {
		case "id":
			id, err := v.StringBytes()
			if err != nil {
				visiterr = fmt.Errorf("invalid 'id' field: %w", err)
			}
			evt.ID = string(id)
		case "pubkey":
			id, err := v.StringBytes()
			if err != nil {
				visiterr = fmt.Errorf("invalid 'pubkey' field: %w", err)
			}
			evt.PubKey = string(id)
		case "created_at":
			val, err := v.Int64()
			if err != nil {
				visiterr = fmt.Errorf("invalid 'created_at' field: %w", err)
			}
			evt.CreatedAt = time.Unix(val, 0)
		case "kind":
			kind, err := v.Int64()
			if err != nil {
				visiterr = fmt.Errorf("invalid 'kind' field: %w", err)
			}
			evt.Kind = int(kind)
		case "tags":
			evt.Tags, err = fastjsonArrayToTags(v)
			if err != nil {
				visiterr = fmt.Errorf("invalid '%s' field: %w", key, err)
			}
		case "content":
			id, err := v.StringBytes()
			if err != nil {
				visiterr = fmt.Errorf("invalid 'content' field: %w", err)
			}
			evt.Content = string(id)
		case "sig":
			id, err := v.StringBytes()
			if err != nil {
				visiterr = fmt.Errorf("invalid 'sig' field: %w", err)
			}
			evt.Sig = string(id)
		default:
			var anyValue any
			json.Unmarshal(v.MarshalTo([]byte{}), &anyValue)
			evt.extra[key] = anyValue
		}
	})
	return visiterr
}

// unmarshaling helper
func fastjsonArrayToTags(v *fastjson.Value) (Tags, error) {
	arr, err := v.Array()
	if err != nil {
		return nil, err
	}

	sll := make(Tags, len(arr))
	for i, v := range arr {
		subarr, err := v.Array()
		if err != nil {
			return nil, err
		}

		sl := make(Tag, len(subarr))
		for j, subv := range subarr {
			sb, err := subv.StringBytes()
			if err != nil {
				return nil, err
			}
			sl[j] = string(sb)
		}
		sll[i] = sl
	}

	return sll, nil
}

// MarshalJSON() returns the JSON byte encoding of the event, as in NIP-01.
func (evt Event) MarshalJSON() ([]byte, error) {
	dst := make([]byte, 0)
	dst = append(dst, '{')
	dst = append(dst, []byte(fmt.Sprintf("\"id\":\"%s\",\"pubkey\":\"%s\",\"created_at\":%d,\"kind\":%d,\"tags\":",
		evt.ID,
		evt.PubKey,
		evt.CreatedAt.Unix(),
		evt.Kind,
	))...)
	dst = evt.Tags.MarshalTo(dst)
	dst = append(dst, []byte(",\"content\":")...)
	dst = escapeString(dst, evt.Content)
	dst = append(dst, []byte(fmt.Sprintf(",\"sig\":\"%s\"",
		evt.Sig,
	))...)
	// slower marshaling of "any" interface type
	if evt.extra != nil {
		buf := bytes.NewBuffer(nil)
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		for k, v := range evt.extra {
			if e := enc.Encode(v); e == nil {
				dst = append(dst, ',')
				dst = escapeString(dst, k)
				dst = append(dst, ':')
				dst = append(dst, buf.Bytes()[:buf.Len()-1]...)
			}
			buf.Reset()
		}
	}
	dst = append(dst, '}')
	return dst, nil
}
