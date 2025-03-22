package nip27

import (
	"iter"
	"regexp"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

type Reference struct {
	Text    string
	Start   int
	End     int
	Pointer nostr.Pointer
}

var mentionRegex = regexp.MustCompile(`\bnostr:((note|npub|naddr|nevent|nprofile)1\w+)\b`)

// Deprecated: this is useless, use Parse() isntead (but the semantics is different)
func ParseReferences(evt nostr.Event) iter.Seq[Reference] {
	return func(yield func(Reference) bool) {
		for _, ref := range mentionRegex.FindAllStringSubmatchIndex(evt.Content, -1) {
			reference := Reference{
				Text:  evt.Content[ref[0]:ref[1]],
				Start: ref[0],
				End:   ref[1],
			}

			nip19code := evt.Content[ref[2]:ref[3]]

			if prefix, data, err := nip19.Decode(nip19code); err == nil {
				switch prefix {
				case "npub":
					pointer := nostr.ProfilePointer{
						PublicKey: data.(string), Relays: []string{},
					}
					tag := evt.Tags.FindWithValue("p", pointer.PublicKey)
					if tag != nil && len(tag) >= 3 {
						pointer.Relays = []string{tag[2]}
					}
					if nostr.IsValidPublicKey(pointer.PublicKey) {
						reference.Pointer = pointer
					}
				case "nprofile":
					pointer := data.(nostr.ProfilePointer)
					tag := evt.Tags.FindWithValue("p", pointer.PublicKey)
					if tag != nil && len(tag) >= 3 {
						pointer.Relays = append(pointer.Relays, tag[2])
					}
					if nostr.IsValidPublicKey(pointer.PublicKey) {
						reference.Pointer = pointer
					}
				case "note":
					// we don't even bother here because people using note1 codes aren't including relay hints anyway
					reference.Pointer = nostr.EventPointer{ID: data.(string), Relays: nil}
				case "nevent":
					pointer := data.(nostr.EventPointer)
					tag := evt.Tags.FindWithValue("e", pointer.ID)
					if tag != nil && len(tag) >= 3 {
						pointer.Relays = append(pointer.Relays, tag[2])
						if pointer.Author == "" && len(tag) >= 5 && nostr.IsValidPublicKey(tag[4]) {
							pointer.Author = tag[4]
						}
					}
					reference.Pointer = pointer
				case "naddr":
					pointer := data.(nostr.EntityPointer)
					tag := evt.Tags.FindWithValue("a", pointer.AsTagReference())
					if tag != nil && len(tag) >= 3 {
						pointer.Relays = append(pointer.Relays, tag[2])
					}
					reference.Pointer = pointer
				}
			}

			if !yield(reference) {
				return
			}
		}
	}
}
