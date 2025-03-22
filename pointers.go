package nostr

import (
	"fmt"
	"strconv"
	"strings"
)

// Pointer is an interface for different types of Nostr pointers.
//
// In this context, a "pointer" is a reference to an event or profile potentially including
// relays and other metadata that might help find it.
type Pointer interface {
	// AsTagReference returns the pointer as a string as it would be seen in the value of a tag (i.e. the tag's second item).
	AsTagReference() string

	// AsTag converts the pointer with all the information available to a tag that can be included in events.
	AsTag() Tag

	// AsFilter converts the pointer to a Filter that can be used to query for it on relays.
	AsFilter() Filter
	MatchesEvent(Event) bool
}

var (
	_ Pointer = (*ProfilePointer)(nil)
	_ Pointer = (*EventPointer)(nil)
	_ Pointer = (*EntityPointer)(nil)
)

// ProfilePointer represents a pointer to a Nostr profile.
type ProfilePointer struct {
	PublicKey string   `json:"pubkey"`
	Relays    []string `json:"relays,omitempty"`
}

// ProfilePointerFromTag creates a ProfilePointer from a "p" tag (but it doesn't have to be necessarily a "p" tag, could be something else).
func ProfilePointerFromTag(refTag Tag) (ProfilePointer, error) {
	pk := refTag[1]
	if !IsValidPublicKey(pk) {
		return ProfilePointer{}, fmt.Errorf("invalid pubkey '%s'", pk)
	}

	pointer := ProfilePointer{
		PublicKey: pk,
	}
	if len(refTag) > 2 {
		if relay := (refTag)[2]; IsValidRelayURL(relay) {
			pointer.Relays = []string{relay}
		}
	}
	return pointer, nil
}

// MatchesEvent checks if the pointer matches an event.
func (ep ProfilePointer) MatchesEvent(_ Event) bool { return false }
func (ep ProfilePointer) AsTagReference() string    { return ep.PublicKey }
func (ep ProfilePointer) AsFilter() Filter          { return Filter{Authors: []string{ep.PublicKey}} }

func (ep ProfilePointer) AsTag() Tag {
	if len(ep.Relays) > 0 {
		return Tag{"p", ep.PublicKey, ep.Relays[0]}
	}
	return Tag{"p", ep.PublicKey}
}

// EventPointer represents a pointer to a nostr event.
type EventPointer struct {
	ID     string   `json:"id"`
	Relays []string `json:"relays,omitempty"`
	Author string   `json:"author,omitempty"`
	Kind   int      `json:"kind,omitempty"`
}

// EventPointerFromTag creates an EventPointer from an "e" tag (but it could be other tag name, it isn't checked).
func EventPointerFromTag(refTag Tag) (EventPointer, error) {
	id := refTag[1]
	if !IsValid32ByteHex(id) {
		return EventPointer{}, fmt.Errorf("invalid id '%s'", id)
	}

	pointer := EventPointer{
		ID: id,
	}
	if len(refTag) > 2 {
		if relay := (refTag)[2]; IsValidRelayURL(relay) {
			pointer.Relays = []string{relay}
		}
		if len(refTag) > 3 && IsValidPublicKey(refTag[3]) {
			pointer.Author = (refTag)[3]
		} else if len(refTag) > 4 && IsValidPublicKey(refTag[4]) {
			pointer.Author = (refTag)[4]
		}
	}
	return pointer, nil
}

func (ep EventPointer) MatchesEvent(evt Event) bool { return evt.ID == ep.ID }
func (ep EventPointer) AsTagReference() string      { return ep.ID }
func (ep EventPointer) AsFilter() Filter            { return Filter{IDs: []string{ep.ID}} }

// AsTag converts the pointer to a Tag.
func (ep EventPointer) AsTag() Tag {
	if len(ep.Relays) > 0 {
		if ep.Author != "" {
			return Tag{"e", ep.ID, ep.Relays[0], ep.Author}
		} else {
			return Tag{"e", ep.ID, ep.Relays[0]}
		}
	}
	return Tag{"e", ep.ID}
}

// EntityPointer represents a pointer to a nostr entity (addressable event).
type EntityPointer struct {
	PublicKey  string   `json:"pubkey"`
	Kind       int      `json:"kind,omitempty"`
	Identifier string   `json:"identifier,omitempty"`
	Relays     []string `json:"relays,omitempty"`
}

// EntityPointerFromTag creates an EntityPointer from an "a" tag (but it doesn't check if the tag is really "a", it could be anything).
func EntityPointerFromTag(refTag Tag) (EntityPointer, error) {
	spl := strings.SplitN(refTag[1], ":", 3)
	if len(spl) != 3 {
		return EntityPointer{}, fmt.Errorf("invalid addr ref '%s'", refTag[1])
	}
	if !IsValidPublicKey(spl[1]) {
		return EntityPointer{}, fmt.Errorf("invalid addr pubkey '%s'", spl[1])
	}

	kind, err := strconv.Atoi(spl[0])
	if err != nil || kind > (1<<16) {
		return EntityPointer{}, fmt.Errorf("invalid addr kind '%s'", spl[0])
	}

	pointer := EntityPointer{
		Kind:       kind,
		PublicKey:  spl[1],
		Identifier: spl[2],
	}
	if len(refTag) > 2 {
		if relay := (refTag)[2]; IsValidRelayURL(relay) {
			pointer.Relays = []string{relay}
		}
	}

	return pointer, nil
}

// MatchesEvent checks if the pointer matches an event.
func (ep EntityPointer) MatchesEvent(evt Event) bool {
	return ep.PublicKey == evt.PubKey &&
		ep.Kind == evt.Kind &&
		evt.Tags.GetD() == ep.Identifier
}

func (ep EntityPointer) AsTagReference() string {
	return fmt.Sprintf("%d:%s:%s", ep.Kind, ep.PublicKey, ep.Identifier)
}

func (ep EntityPointer) AsFilter() Filter {
	return Filter{
		Kinds:   []int{ep.Kind},
		Authors: []string{ep.PublicKey},
		Tags:    TagMap{"d": []string{ep.Identifier}},
	}
}

func (ep EntityPointer) AsTag() Tag {
	if len(ep.Relays) > 0 {
		return Tag{"a", ep.AsTagReference(), ep.Relays[0]}
	}
	return Tag{"a", ep.AsTagReference()}
}
