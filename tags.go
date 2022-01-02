package nostr

import (
	"encoding/json"
	"errors"
)

type Tags []Tag
type Tag []interface{}

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

		currentTagName, ok := tag[0].(string)
		if !ok || currentTagName != tagName {
			continue
		}

		currentTagValue, ok := tag[1].(string)
		if !ok {
			continue
		}

		if values.Contains(currentTagValue) {
			return true
		}
	}

	return false
}
