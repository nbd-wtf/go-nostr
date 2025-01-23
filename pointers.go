package nostr

import (
	"fmt"
)

type Pointer interface {
	AsTagReference() string
	AsTag() Tag
	MatchesEvent(Event) bool
}

type ProfilePointer struct {
	PublicKey string   `json:"pubkey"`
	Relays    []string `json:"relays,omitempty"`
}

func (ep ProfilePointer) MatchesEvent(_ Event) bool { return false }
func (ep ProfilePointer) AsTagReference() string    { return ep.PublicKey }

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

func (ep EventPointer) MatchesEvent(evt Event) bool { return evt.ID == ep.ID }
func (ep EventPointer) AsTagReference() string      { return ep.ID }

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

func (ep EntityPointer) MatchesEvent(evt Event) bool {
	return ep.PublicKey == evt.PubKey &&
		ep.Kind == evt.Kind &&
		evt.Tags.GetD() == ep.Identifier
}

func (ep EntityPointer) AsTagReference() string {
	return fmt.Sprintf("%d:%s:%s", ep.Kind, ep.PublicKey, ep.Identifier)
}

func (ep EntityPointer) AsTag() Tag {
	if len(ep.Relays) > 0 {
		return Tag{"a", ep.AsTagReference(), ep.Relays[0]}
	}
	return Tag{"a", ep.AsTagReference()}
}
