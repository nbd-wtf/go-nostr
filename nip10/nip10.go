package nip10

import "github.com/nbd-wtf/go-nostr"

func GetThreadRoot(tags nostr.Tags) nostr.Pointer {
	for _, tag := range tags {
		if len(tag) >= 4 && tag[0] == "e" && tag[3] == "root" {
			p, _ := nostr.EventPointerFromTag(tag)
			return p
		}
	}

	firstE := tags.Find("e")
	if firstE != nil {
		return nostr.EventPointer{
			ID: firstE[1],
		}
	}

	return nil
}

func GetImmediateParent(tags nostr.Tags) nostr.Pointer {
	var parent nostr.Tag
	var lastE nostr.Tag

	for i := 0; i <= len(tags)-1; i++ {
		tag := tags[i]

		if len(tag) < 2 {
			continue
		}
		if tag[0] != "e" && tag[0] != "a" {
			continue
		}

		if len(tag) >= 4 {
			if tag[3] == "reply" {
				parent = tag
				break
			}
			if tag[3] == "parent" {
				// will be used as our first fallback
				parent = tag
				continue
			}
			if tag[3] == "mention" {
				// this invalidates this tag as a second fallback mechanism (clients that don't add markers)
				continue
			}
		}

		lastE = tag // will be used as our second fallback (clients that don't add markers)
	}

	// if we reached this point we don't have a "reply", but if we have a "parent"
	// that means this event is a direct reply to the parent
	if parent != nil {
		p, _ := nostr.EventPointerFromTag(parent)
		return p
	}

	if lastE != nil {
		// if we reached this point and we have at least one "e" we'll use that (the last)
		// (we don't bother looking for relay or author hints because these clients don't add these anyway)
		return nostr.EventPointer{
			ID: lastE[1],
		}
	}

	return nil
}
