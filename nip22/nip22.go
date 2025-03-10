package nip22

import "github.com/nbd-wtf/go-nostr"

func GetThreadRoot(tags nostr.Tags) nostr.Pointer {
	for _, tag := range tags {
		if len(tag) < 2 {
			continue
		}
		switch tag[0] {
		case "E":
			ep, _ := nostr.EventPointerFromTag(tag)
			return ep
		case "A":
			ep, _ := nostr.EntityPointerFromTag(tag)
			return ep
		case "I":
			ep, _ := nostr.ExternalPointerFromTag(tag)
			return ep
		}
	}
	return nil
}

func GetImmediateParent(tags nostr.Tags) nostr.Pointer {
	for _, tag := range tags {
		if len(tag) < 2 {
			continue
		}
		switch tag[0] {
		case "e":
			ep, _ := nostr.EventPointerFromTag(tag)
			return ep
		case "a":
			ep, _ := nostr.EntityPointerFromTag(tag)
			return ep
		case "i":
			ep, _ := nostr.ExternalPointerFromTag(tag)
			return ep
		}
	}
	return nil
}
