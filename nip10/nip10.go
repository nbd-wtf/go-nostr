package nip10

import "github.com/nbd-wtf/go-nostr"

func GetThreadRoot(tags nostr.Tags) *nostr.Tag {
	for _, tag := range tags {
		if len(tag) >= 4 && tag[0] == "e" && tag[3] == "root" {
			return &tag
		}
	}

	return tags.GetFirst([]string{"e", ""})
}

func GetImmediateReply(tags nostr.Tags) *nostr.Tag {
	for i := len(tags) - 1; i >= 0; i-- {
		tag := tags[i]
		if len(tag) >= 4 && tag[0] == "e" && tag[3] == "reply" {
			return &tag
		}
	}

	return tags.GetLast([]string{"e", ""})
}
