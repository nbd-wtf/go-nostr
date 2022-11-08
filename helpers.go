package nostr

import (
	"strings"

	"golang.org/x/exp/constraints"
)

func Similar[E constraints.Ordered](as, bs []E) bool {
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

func ContainsPrefixOf(haystack []string, needle string) bool {
	for _, hay := range haystack {
		if strings.HasPrefix(needle, hay) {
			return true
		}
	}
	return false
}
