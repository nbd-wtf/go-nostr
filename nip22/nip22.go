package nip22

import "github.com/nbd-wtf/go-nostr"

func GetThreadRoot(tags nostr.Tags) *nostr.Tag {
	for _, tag := range tags {
		if tag[0] == "E" || tag[0] == "A" || tag[0] == "I" {
			return &tag
		}
	}
	empty := nostr.Tag{}
	return &empty
}

func GetImmediateReply(tags nostr.Tags) *nostr.Tag {
	for _, tag := range tags {
		if tag[0] == "e" || tag[0] == "a" || tag[0] == "i" {
			return &tag
		}
	}

	empty := nostr.Tag{}
	return &empty
}
