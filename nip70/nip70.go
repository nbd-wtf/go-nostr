package nip70

import "github.com/nbd-wtf/go-nostr"

func IsProtected(event *nostr.Event) bool {
	for _, tag := range event.Tags {
		if len(tag) == 1 && tag[0] == "-" {
			return true
		}
	}
	return false
}
