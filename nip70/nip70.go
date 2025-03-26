package nip70

import (
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

func IsProtected(event nostr.Event) bool {
	for _, tag := range event.Tags {
		if len(tag) == 1 && tag[0] == "-" {
			return true
		}
	}
	return false
}

func HasEmbeddedProtected(event nostr.Event) bool {
	if event.Kind == 6 || event.Kind == 16 {
		tidx := strings.Index(event.Content, `"tags":[`)
		eidx := strings.Index(event.Content, `]]`)
		pidx := strings.Index(event.Content, `["-"]`)
		if tidx != -1 && eidx != -1 && pidx != -1 {
			return pidx > tidx && pidx < eidx
		}
	}

	return false
}
