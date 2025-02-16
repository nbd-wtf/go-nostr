package nostr

import (
	"fmt"
	"strconv"
	"strings"
)

type Pointer interface {
	AsTagReference() string
	AsTag() Tag
	AsFilter() Filter
	MatchesEvent(Event) bool
}

var (
	_ Pointer = (*ProfilePointer)(nil)
	_ Pointer = (*EventPointer)(nil)
	_ Pointer = (*EntityPointer)(nil)
)

type ProfilePointer struct {
	PublicKey string   `json:"pubkey"`
	Relays    []string `json:"relays,omitempty"`
}

func ProfilePointerFromTag(refTag Tag) (ProfilePointer, error) {
	pk := (refTag)[1]
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

func (ep ProfilePointer) MatchesEvent(_ Event) bool { return false }
func (ep ProfilePointer) AsTagReference() string    { return ep.PublicKey }
func (ep ProfilePointer) AsFilter() Filter          { return Filter{Authors: []string{ep.PublicKey}} }

func (ep ProfilePointer) AsTag() Tag {
	if len(ep.Relays) > 0 {
		return Tag{"p", ep.PublicKey, ep.Relays[0]}
	}
	return Tag{"p", ep.PublicKey}
}

type EventPointer struct {
	ID     string   `json:"id"`
	Relays []string `json:"relays,omitempty"`
	Author string   `json:"author,omitempty"`
	Kind   int      `json:"kind,omitempty"`
}

func EventPointerFromTag(refTag Tag) (EventPointer, error) {
	id := (refTag)[1]
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
		if len(refTag) > 3 && IsValidPublicKey((refTag)[3]) {
			pointer.Author = (refTag)[3]
		}
	}
	return pointer, nil
}

func (ep EventPointer) MatchesEvent(evt Event) bool { return evt.ID == ep.ID }
func (ep EventPointer) AsTagReference() string      { return ep.ID }
func (ep EventPointer) AsFilter() Filter            { return Filter{IDs: []string{ep.ID}} }

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

type EntityPointer struct {
	PublicKey  string   `json:"pubkey"`
	Kind       int      `json:"kind,omitempty"`
	Identifier string   `json:"identifier,omitempty"`
	Relays     []string `json:"relays,omitempty"`
}

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
