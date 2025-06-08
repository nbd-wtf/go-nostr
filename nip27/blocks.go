package nip27

import (
	"iter"
	"net/url"
	"regexp"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/go-nostr/nip73"
)

type Block struct {
	Text    string
	Start   int
	Pointer nostr.Pointer
}

var (
	noCharacter    = regexp.MustCompile(`(?m)\W`)
	noURLCharacter = regexp.MustCompile(`(?m)\W |\W$|$|,| `)
)

func Parse(content string) iter.Seq[Block] {
	return func(yield func(Block) bool) {
		max := len(content)
		index := 0
		prevIndex := 0

		for index < max {
			pu := strings.IndexRune(content[index:], ':')
			if pu == -1 {
				// reached end
				break
			}
			u := pu + index

			switch {
			case u >= 5 && content[u-5:u] == "nostr" && u+60 < max:
				m := noCharacter.FindStringIndex(content[u+60:])
				end := max
				if m != nil {
					end = u + 60 + m[0]
				}

				prefix, data, err := nip19.Decode(content[u+1 : end])
				if err != nil {
					// ignore this, not a valid nostr uri
					index = u + 1
					continue
				}

				var pointer nostr.Pointer
				switch prefix {
				case "npub":
					pointer = nostr.ProfilePointer{PublicKey: data.(string)}
				case "nprofile", "nevent", "naddr":
					pointer = data.(nostr.Pointer)
				case "note", "nsec":
					fallthrough // I'm so cool
				default:
					// ignore this, treat it as not a valid uri
					index = end + 1
					continue
				}

				if prevIndex != u-5 {
					if !yield(Block{Text: content[prevIndex : u-5], Start: prevIndex}) {
						return
					}
				}

				if !yield(Block{Pointer: pointer, Text: content[u-5 : end], Start: u - 5}) {
					return
				}

				index = end
				prevIndex = index
				continue
			case ((u >= 5 && content[u-5:u] == "https") || (u >= 4 && content[u-4:u] == "http")) && u+4 < max:
				m := noURLCharacter.FindStringIndex(content[u+4:])
				end := max
				if m != nil {
					end = u + 4 + m[0]
				}
				prefixLen := 4
				if content[u-1] == 's' {
					prefixLen = 5
				}
				parsed, err := url.Parse(content[u-prefixLen : end])
				if err != nil || !strings.Contains(parsed.Host, ".") {
					// ignore this, not a valid url
					index = end + 1
					continue
				}

				if prevIndex != u-prefixLen {
					if !yield(Block{Text: content[prevIndex : u-prefixLen], Start: prevIndex}) {
						return
					}
				}

				if !yield(Block{Pointer: nip73.ExternalPointer{Thing: content[u-prefixLen : end]}, Text: content[u-prefixLen : end], Start: u - prefixLen}) {
					return
				}

				index = end
				prevIndex = index
				continue
			case ((u >= 3 && content[u-3:u] == "wss") || (u >= 2 && content[u-2:u] == "ws")) && u+4 < max:
				m := noURLCharacter.FindStringIndex(content[u+4:])
				end := max
				if m != nil {
					end = u + 4 + m[0]
				}
				prefixLen := 2
				if content[u-2] == 's' {
					prefixLen = 3
				}
				parsed, err := url.Parse(content[u-prefixLen : end])
				if err != nil || !strings.Contains(parsed.Host, ".") {
					// ignore this, not a valid url
					index = end + 1
					continue
				}

				if prevIndex != u-prefixLen {
					if !yield(Block{Text: content[prevIndex : u-prefixLen], Start: prevIndex}) {
						return
					}
				}

				if !yield(Block{Pointer: nip73.ExternalPointer{Thing: content[u-prefixLen : end]}, Text: content[u-prefixLen : end], Start: u - prefixLen}) {
					return
				}

				index = end
				prevIndex = index
				continue
			default:
				// ignore this, it is nothing
				index = u + 1
				continue
			}
		}

		if prevIndex != max {
			yield(Block{Text: content[prevIndex:], Start: prevIndex})
		}
	}
}
