package nostr

import (
	"strings"

	"golang.org/x/exp/constraints"
)

func similar[E constraints.Ordered](as, bs []E) bool {
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

func containsPrefixOf(haystack []string, needle string) bool {
	for _, hay := range haystack {
		if strings.HasPrefix(needle, hay) {
			return true
		}
	}
	return false
}

// Escaping strings for JSON encoding according to RFC8259.
// Also encloses result in quotation marks "".
func escapeString(dst []byte, s string) []byte {
	dst = append(dst, '"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			// quotation mark
			dst = append(dst, []byte{'\\', '"'}...)
		case c == '\\':
			// reverse solidus
			dst = append(dst, []byte{'\\', '\\'}...)
		case c >= 0x20:
			// default, rest below are control chars
			dst = append(dst, c)
		case c == 0x08:
			dst = append(dst, []byte{'\\', 'b'}...)
		case c < 0x09:
			dst = append(dst, []byte{'\\', 'u', '0', '0', '0', '0' + c}...)
		case c == 0x09:
			dst = append(dst, []byte{'\\', 't'}...)
		case c == 0x0a:
			dst = append(dst, []byte{'\\', 'n'}...)
		case c == 0x0c:
			dst = append(dst, []byte{'\\', 'f'}...)
		case c == 0x0d:
			dst = append(dst, []byte{'\\', 'r'}...)
		case c < 0x10:
			dst = append(dst, []byte{'\\', 'u', '0', '0', '0', 0x57 + c}...)
		case c < 0x1a:
			dst = append(dst, []byte{'\\', 'u', '0', '0', '1', 0x20 + c}...)
		case c < 0x20:
			dst = append(dst, []byte{'\\', 'u', '0', '0', '1', 0x47 + c}...)
		}
	}
	dst = append(dst, '"')
	return dst
}

func InsertEventIntoDescendingList(sortedArray []*Event, event *Event) []*Event {
	size := len(sortedArray)
	start := 0
	end := size - 1
	var mid int
	position := start

	if end < 0 {
		return []*Event{event}
	} else if event.CreatedAt < sortedArray[end].CreatedAt {
		return append(sortedArray, event)
	} else if event.CreatedAt > sortedArray[start].CreatedAt {
		newArr := make([]*Event, size+1)
		newArr[0] = event
		copy(newArr[1:], sortedArray)
		return newArr
	} else if event.CreatedAt == sortedArray[start].CreatedAt {
		position = start
	} else {
		for {
			if end <= start+1 {
				position = end
				break
			}
			mid = int(start + (end-start)/2)
			if sortedArray[mid].CreatedAt > event.CreatedAt {
				start = mid
			} else if sortedArray[mid].CreatedAt < event.CreatedAt {
				end = mid
			} else {
				position = mid
				break
			}
		}
	}

	if sortedArray[position].ID != event.ID {
		if cap(sortedArray) > size {
			newArr := sortedArray[0 : size+1]
			copy(newArr[position+1:], sortedArray[position:])
			newArr[position] = event
			return newArr
		} else {
			newArr := make([]*Event, size+1, size+5)
			copy(newArr[:position], sortedArray[:position])
			copy(newArr[position+1:], sortedArray[position:])
			newArr[position] = event
			return newArr
		}
	}

	return sortedArray
}
