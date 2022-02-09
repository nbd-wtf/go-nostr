package nostr

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/valyala/fastjson"
)

type Tags []StringList

func (t *Tags) Scan(src interface{}) error {
	var jtags []byte = make([]byte, 0)

	switch v := src.(type) {
	case []byte:
		jtags = v
	case string:
		jtags = []byte(v)
	default:
		return errors.New("couldn't scan tags, it's not a json string")
	}

	json.Unmarshal(jtags, &t)
	return nil
}

func (tags Tags) ContainsAny(tagName string, values StringList) bool {
	for _, tag := range tags {
		if len(tag) < 2 {
			continue
		}

		if tag[0] != tagName {
			continue
		}

		if values.Contains(tag[1]) {
			return true
		}
	}

	return false
}

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
		}
	})
	if visiterr != nil {
		return visiterr
	}

	return nil
}

func (evt Event) MarshalJSON() ([]byte, error) {
	var arena fastjson.Arena

	o := arena.NewObject()
	o.Set("id", arena.NewString(evt.ID))
	o.Set("pubkey", arena.NewString(evt.PubKey))
	o.Set("created_at", arena.NewNumberInt(int(evt.CreatedAt.Unix())))
	o.Set("kind", arena.NewNumberInt(evt.Kind))
	o.Set("tags", tagsToFastjsonArray(&arena, evt.Tags))
	o.Set("content", arena.NewString(evt.Content))
	o.Set("sig", arena.NewString(evt.Sig))

	return o.MarshalTo(nil), nil
}

func fastjsonArrayToTags(v *fastjson.Value) (Tags, error) {
	arr, err := v.Array()
	if err != nil {
		return nil, err
	}

	sll := make([]StringList, len(arr))
	for i, v := range arr {
		subarr, err := v.Array()
		if err != nil {
			return nil, err
		}

		sl := make(StringList, len(arr))
		for j, subv := range subarr {
			sb, err := subv.StringBytes()
			if err != nil {
				return nil, err
			}
			sl[j] = string(sb)
		}
		sll[i] = sl
	}

	return Tags(sll), nil
}

func tagsToFastjsonArray(arena *fastjson.Arena, tags Tags) *fastjson.Value {
	jtags := arena.NewArray()
	for i, v := range tags {
		arr := arena.NewArray()
		for j, subv := range v {
			arr.SetArrayItem(j, arena.NewString(subv))
		}
		jtags.SetArrayItem(i, arr)
	}
	return jtags
}
