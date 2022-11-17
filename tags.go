package nostr

import (
	"encoding/json"
	"errors"
	"strings"

	"golang.org/x/exp/slices"
)

type Tag []string

// StartsWith checks if a tag contains a prefix.
// for example,
//     ["p", "abcdef...", "wss://relay.com"]
// would match against
//     ["p", "abcdef..."]
// or even
//     ["p", "abcdef...", "wss://"]
func (tag Tag) StartsWith(prefix []string) bool {
	prefixLen := len(prefix)

	if prefixLen > len(tag) {
		return false
	}
	// check initial elements for equality
	for i := 0; i < prefixLen-1; i++ {
		if prefix[i] != tag[i] {
			return false
		}
	}
	// check last element just for a prefix
	return strings.HasPrefix(tag[prefixLen-1], prefix[prefixLen-1])
}

func (tag Tag) Key() string {
	if len(tag) > 0 {
		return tag[0]
	}
	return ""
}

func (tag Tag) Value() string {
	if len(tag) > 1 {
		return tag[1]
	}
	return ""
}

func (tag Tag) Relay() string {
	if tag[0] == "e" || tag[0] == "p" && len(tag) > 2 {
		return tag[2]
	}
	return ""
}

type Tags []Tag

// GetFirst gets the first tag in tags that matches the prefix, see [Tag.StartsWith]
func (tags Tags) GetFirst(tagPrefix []string) *Tag {
	for _, v := range tags {
		if v.StartsWith(tagPrefix) {
			return &v
		}
	}
	return nil
}

// GetLast gets the last tag in tags that matches the prefix, see [Tag.StartsWith]
func (tags Tags) GetLast(tagPrefix []string) *Tag {
	for i := len(tags) - 1; i >= 0; i-- {
		v := tags[i]
		if v.StartsWith(tagPrefix) {
			return &v
		}
	}
	return nil
}

// GetLast gets all the tags that match the prefix, see [Tag.StartsWith]
func (tags Tags) GetAll(tagPrefix []string) Tags {
	result := make(Tags, 0, len(tags))
	for _, v := range tags {
		if v.StartsWith(tagPrefix) {
			result = append(result, v)
		}
	}
	return result
}

// FilterOut removes all tags that match the prefix, see [Tag.StartsWith]
func (tags Tags) FilterOut(tagPrefix []string) Tags {
	filtered := make(Tags, 0, len(tags))
	for _, v := range tags {
		if !v.StartsWith(tagPrefix) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// AppendUnique appends a tag if it doesn't exist yet, otherwise does nothing
func (tags Tags) AppendUnique(tag Tag) Tags {
	if tags.GetFirst(tag) == nil {
		return append(tags, tag)
	} else {
		return tags
	}
}

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

func (tags Tags) ContainsAny(tagName string, values []string) bool {
	for _, tag := range tags {
		if len(tag) < 2 {
			continue
		}

		if tag[0] != tagName {
			continue
		}

		if slices.Contains(values, tag[1]) {
			return true
		}
	}

	return false
}
