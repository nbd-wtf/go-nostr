package filter

import "github.com/fiatjaf/go-nostr/event"

func stringsEqual(as, bs []string) bool {
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

func intsEqual(as, bs []int) bool {
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

func stringsContain(haystack []string, needle string) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}
	return false
}

func intsContain(haystack []int, needle int) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}
	return false
}

func containsAnyTag(tagName string, tags event.Tags, values []string) bool {
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

		if stringsContain(values, currentTagValue) {
			return true
		}
	}

	return false
}
