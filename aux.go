package nostr

import (
	"strings"
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
