package nostr

import (
	"fmt"
	"strings"
	"time"

	"github.com/valyala/fastjson"
)

type StringList []string
type IntList []int

func (as StringList) Equals(bs StringList) bool {
	if len(as) != len(bs) {
		return false
	}

	for _, a := range as {
		for _, b := range bs {
			if b == a {
				goto next
			}
		}
		// didn't find a B that corresponded to the current A
		return false

	next:
		continue
	}

	return true
}

func (as IntList) Equals(bs IntList) bool {
	if len(as) != len(bs) {
		return false
	}

	for _, a := range as {
		for _, b := range bs {
			if b == a {
				goto next
			}
		}
		// didn't find a B that corresponded to the current A
		return false

	next:
		continue
	}

	return true
}

func (haystack StringList) Contains(needle string) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}
	return false
}

func (haystack StringList) ContainsPrefixOf(needle string) bool {
	for _, hay := range haystack {
		if strings.HasPrefix(needle, hay) {
			return true
		}
	}
	return false
}

func (haystack IntList) Contains(needle int) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}
	return false
}

func (f *Filter) UnmarshalJSON(payload []byte) error {
	var fastjsonParser fastjson.Parser
	parsed, err := fastjsonParser.ParseBytes(payload)
	if err != nil {
		return fmt.Errorf("failed to parse filter: %w", err)
	}

	obj, err := parsed.Object()
	if err != nil {
		return fmt.Errorf("filter is not an object")
	}

	f.Tags = make(map[string]StringList)

	var visiterr error
	obj.Visit(func(k []byte, v *fastjson.Value) {
		key := string(k)
		switch key {
		case "ids":
			f.IDs, err = fastjsonArrayToStringList(v)
			if err != nil {
				visiterr = fmt.Errorf("invalid 'ids' field: %w", err)
			}
		case "kinds":
			f.Kinds, err = fastjsonArrayToIntList(v)
			if err != nil {
				visiterr = fmt.Errorf("invalid 'kinds' field: %w", err)
			}
		case "authors":
			f.Authors, err = fastjsonArrayToStringList(v)
			if err != nil {
				visiterr = fmt.Errorf("invalid 'authors' field: %w", err)
			}
		case "since":
			val, err := v.Int64()
			if err != nil {
				visiterr = fmt.Errorf("invalid 'since' field: %w", err)
			}
			tm := time.Unix(val, 0)
			f.Since = &tm
		case "until":
			val, err := v.Int64()
			if err != nil {
				visiterr = fmt.Errorf("invalid 'until' field: %w", err)
			}
			tm := time.Unix(val, 0)
			f.Until = &tm
		default:
			if strings.HasPrefix(key, "#") {
				f.Tags[key[1:]], err = fastjsonArrayToStringList(v)
				if err != nil {
					visiterr = fmt.Errorf("invalid '%s' field: %w", key, err)
				}
			}
		}
	})
	if visiterr != nil {
		return visiterr
	}

	return nil
}

func (f Filter) MarshalJSON() ([]byte, error) {
	var arena fastjson.Arena

	o := arena.NewObject()

	if f.IDs != nil {
		o.Set("ids", stringListToFastjsonArray(&arena, f.IDs))
	}
	if f.Kinds != nil {
		o.Set("kinds", intListToFastjsonArray(&arena, f.Kinds))
	}
	if f.Authors != nil {
		o.Set("authors", stringListToFastjsonArray(&arena, f.Authors))
	}
	if f.Since != nil {
		o.Set("since", arena.NewNumberInt(int(f.Since.Unix())))
	}
	if f.Until != nil {
		o.Set("until", arena.NewNumberInt(int(f.Until.Unix())))
	}
	if f.Tags != nil {
		for k, v := range f.Tags {
			o.Set("#"+k, stringListToFastjsonArray(&arena, v))
		}
	}

	return o.MarshalTo(nil), nil
}

func stringListToFastjsonArray(arena *fastjson.Arena, sl StringList) *fastjson.Value {
	arr := arena.NewArray()
	for i, v := range sl {
		arr.SetArrayItem(i, arena.NewString(v))
	}
	return arr
}

func intListToFastjsonArray(arena *fastjson.Arena, il IntList) *fastjson.Value {
	arr := arena.NewArray()
	for i, v := range il {
		arr.SetArrayItem(i, arena.NewNumberInt(v))
	}
	return arr
}

func fastjsonArrayToStringList(v *fastjson.Value) (StringList, error) {
	arr, err := v.Array()
	if err != nil {
		return nil, err
	}

	sl := make(StringList, len(arr))
	for i, v := range arr {
		sb, err := v.StringBytes()
		if err != nil {
			return nil, err
		}
		sl[i] = string(sb)
	}

	return sl, nil
}

func fastjsonArrayToIntList(v *fastjson.Value) (IntList, error) {
	arr, err := v.Array()
	if err != nil {
		return nil, err
	}

	il := make(IntList, len(arr))
	for i, v := range arr {
		il[i], err = v.Int()
		if err != nil {
			return nil, err
		}
	}

	return il, nil
}
