package sdk

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

type Reference struct {
	Text    string
	Profile *nostr.ProfilePointer
	Event   *nostr.EventPointer
	Entity  *nostr.EntityPointer
}

var mentionRegex = regexp.MustCompile(`\bnostr:((note|npub|naddr|nevent|nprofile)1\w+)\b|#\[(\d+)\]`)

// ParseReferences parses both NIP-08 and NIP-27 references in a single unifying interface.
func ParseReferences(evt *nostr.Event) []*Reference {
	var references []*Reference
	for _, ref := range mentionRegex.FindAllStringSubmatch(evt.Content, -1) {
		if ref[2] != "" {
			// it's a NIP-27 mention
			if prefix, data, err := nip19.Decode(ref[1]); err == nil {
				switch prefix {
				case "npub":
					references = append(references, &Reference{
						Text: ref[0],
						Profile: &nostr.ProfilePointer{
							PublicKey: data.(string), Relays: []string{},
						},
					})
				case "nprofile":
					pp := data.(nostr.ProfilePointer)
					references = append(references, &Reference{
						Text:    ref[0],
						Profile: &pp,
					})
				case "note":
					references = append(references, &Reference{
						Text:  ref[0],
						Event: &nostr.EventPointer{ID: data.(string), Relays: []string{}},
					})
				case "nevent":
					evp := data.(nostr.EventPointer)
					references = append(references, &Reference{
						Text:  ref[0],
						Event: &evp,
					})
				case "naddr":
					addr := data.(nostr.EntityPointer)
					references = append(references, &Reference{
						Text:   ref[0],
						Entity: &addr,
					})
				}
			}
		} else if ref[3] != "" {
			// it's a NIP-10 mention
			idx, err := strconv.Atoi(ref[3])
			if err != nil || len(evt.Tags) <= idx {
				continue
			}
			if tag := evt.Tags[idx]; tag != nil {
				switch tag[0] {
				case "p":
					relays := make([]string, 0, 1)
					if len(tag) > 2 && tag[2] != "" {
						relays = append(relays, tag[2])
					}
					references = append(references, &Reference{
						Text: ref[0],
						Profile: &nostr.ProfilePointer{
							PublicKey: tag[1],
							Relays:    relays,
						},
					})
				case "e":
					relays := make([]string, 0, 1)
					if len(tag) > 2 && tag[2] != "" {
						relays = append(relays, tag[2])
					}
					references = append(references, &Reference{
						Text: ref[0],
						Event: &nostr.EventPointer{
							ID:     tag[1],
							Relays: relays,
						},
					})
				case "a":
					if parts := strings.Split(ref[1], ":"); len(parts) == 3 {
						kind, _ := strconv.Atoi(parts[0])
						relays := make([]string, 0, 1)
						if len(tag) > 2 && tag[2] != "" {
							relays = append(relays, tag[2])
						}
						references = append(references, &Reference{
							Text: ref[0],
							Entity: &nostr.EntityPointer{
								Identifier: parts[2],
								PublicKey:  parts[1],
								Kind:       kind,
								Relays:     relays,
							},
						})
					}
				}
			}
		}
	}

	return references
}
