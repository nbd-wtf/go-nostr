package nip34

import (
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/nbd-wtf/go-nostr"
)

type Patch struct {
	nostr.Event

	Repository nostr.EntityPointer

	Files  []*gitdiff.File
	Header *gitdiff.PatchHeader
}

func ParsePatch(event nostr.Event) Patch {
	patch := Patch{
		Event: event,
	}

	for _, tag := range event.Tags {
		if len(tag) < 2 {
			continue
		}
		switch tag[0] {
		case "a":
			spl := strings.Split(tag[1], ":")
			if len(spl) != 3 {
				continue
			}
			if !nostr.IsValid32ByteHex(spl[1]) {
				continue
			}
			patch.Repository.Kind = nostr.KindRepositoryAnnouncement
			patch.Repository.PublicKey = spl[1]
			patch.Repository.Identifier = spl[2]
			if len(tag) >= 3 {
				patch.Repository.Relays = []string{tag[2]}
			}
		}
	}

	files, preamble, err := gitdiff.Parse(strings.NewReader(event.Content))
	if err != nil {
		return patch
	}
	patch.Files = files

	header, err := gitdiff.ParsePatchHeader(preamble)
	if err != nil {
		return patch
	}
	patch.Header = header

	return patch
}
