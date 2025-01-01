package nip14

import "github.com/nbd-wtf/go-nostr"

func GetSubject(tags nostr.Tags) string {
	for _, tag := range tags {
		if len(tag) >= 2 && tag[0] == "subject" {
			return tag[1]
		}
	}
	return ""
}
