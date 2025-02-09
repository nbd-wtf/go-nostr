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
	Profile *nostr.ProfilePointer
	Event   *nostr.EventPointer
	Entity  *nostr.EntityPointer
}

var mentionRegex = regexp.MustCompile(`\bnostr:((note|npub|naddr|nevent|nprofile)1\w+)\b`)

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
					reference.Profile = &nostr.ProfilePointer{
						PublicKey: data.(string), Relays: []string{},
					}
					tag := evt.Tags.GetFirst([]string{"p", reference.Profile.PublicKey})
					if tag != nil && len(*tag) >= 3 {
						reference.Profile.Relays = []string{(*tag)[2]}
					}
				case "nprofile":
					pp := data.(nostr.ProfilePointer)
					reference.Profile = &pp
					tag := evt.Tags.GetFirst([]string{"p", reference.Profile.PublicKey})
					if tag != nil && len(*tag) >= 3 {
						reference.Profile.Relays = append(reference.Profile.Relays, (*tag)[2])
					}
				case "note":
					// we don't even bother here because people using note1 codes aren't including relay hints anyway
					reference.Event = &nostr.EventPointer{ID: data.(string), Relays: []string{}}
				case "nevent":
					evp := data.(nostr.EventPointer)
					reference.Event = &evp
					tag := evt.Tags.GetFirst([]string{"e", reference.Event.ID})
					if tag != nil && len(*tag) >= 3 {
						reference.Event.Relays = append(reference.Event.Relays, (*tag)[2])
						if reference.Event.Author == "" && len(*tag) >= 5 {
							reference.Event.Author = (*tag)[4]
						}
					}
				case "naddr":
					addr := data.(nostr.EntityPointer)
					reference.Entity = &addr
					tag := evt.Tags.GetFirst([]string{"a", reference.Entity.AsTagReference()})
					if tag != nil && len(*tag) >= 3 {
						reference.Entity.Relays = append(reference.Entity.Relays, (*tag)[2])
					}
				}
			}

			if !yield(reference) {
				return
			}
		}
	}
}
