package sdk

import (
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip10"
	"github.com/nbd-wtf/go-nostr/nip22"
)

func GetThreadRoot(evt *nostr.Event) *nostr.Tag {
	if evt.Kind == nostr.KindComment {
		return nip22.GetThreadRoot(evt.Tags)
	} else {
		return nip10.GetThreadRoot(evt.Tags)
	}
}

func GetImmediateReply(evt *nostr.Event) *nostr.Tag {
	if evt.Kind == nostr.KindComment {
		return nip22.GetImmediateReply(evt.Tags)
	} else {
		return nip10.GetImmediateReply(evt.Tags)
	}
}
