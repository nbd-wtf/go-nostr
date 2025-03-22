package nip73

import "github.com/nbd-wtf/go-nostr"

var _ nostr.Pointer = (*ExternalPointer)(nil)

// ExternalPointer represents a pointer to a URL or something else.
type ExternalPointer struct {
	Thing string
}

// ExternalPointerFromTag creates a ExternalPointer from an "i" tag
func ExternalPointerFromTag(refTag nostr.Tag) (ExternalPointer, error) {
	return ExternalPointer{refTag[1]}, nil
}

func (ep ExternalPointer) MatchesEvent(_ nostr.Event) bool { return false }
func (ep ExternalPointer) AsTagReference() string          { return ep.Thing }
func (ep ExternalPointer) AsFilter() nostr.Filter          { return nostr.Filter{} }

func (ep ExternalPointer) AsTag() nostr.Tag {
	return nostr.Tag{"i", ep.Thing}
}
