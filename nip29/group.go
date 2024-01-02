package nip29

import (
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

type Group struct {
	ID      string
	Name    string
	Picture string
	About   string
	Members map[string]*Role
	Private bool
	Closed  bool

	LastMetadataUpdate nostr.Timestamp
}

func (group Group) ToMetadataEvent() *nostr.Event {
	evt := &nostr.Event{
		Kind:      39000,
		CreatedAt: group.LastMetadataUpdate,
		Content:   group.About,
		Tags: nostr.Tags{
			nostr.Tag{"d", group.ID},
		},
	}
	if group.Name != "" {
		evt.Tags = append(evt.Tags, nostr.Tag{"name", group.Name})
	}
	if group.About != "" {
		evt.Tags = append(evt.Tags, nostr.Tag{"about", group.Name})
	}
	if group.Picture != "" {
		evt.Tags = append(evt.Tags, nostr.Tag{"picture", group.Picture})
	}

	// status
	if group.Private {
		evt.Tags = append(evt.Tags, nostr.Tag{"private"})
	} else {
		evt.Tags = append(evt.Tags, nostr.Tag{"public"})
	}
	if group.Closed {
		evt.Tags = append(evt.Tags, nostr.Tag{"closed"})
	} else {
		evt.Tags = append(evt.Tags, nostr.Tag{"open"})
	}

	return evt
}

func (group *Group) MergeInMetadataEvent(evt *nostr.Event) error {
	if evt.Kind != nostr.KindSimpleGroupMetadata {
		return fmt.Errorf("expected kind %d, got %d", nostr.KindSimpleGroupMetadata, evt.Kind)
	}

	if evt.CreatedAt <= group.LastMetadataUpdate {
		return fmt.Errorf("event is older than our last update (%d vs %d)", evt.CreatedAt, group.LastMetadataUpdate)
	}

	group.LastMetadataUpdate = evt.CreatedAt
	group.ID = evt.Tags.GetD()
	group.Name = group.ID

	if tag := evt.Tags.GetFirst([]string{"name", ""}); tag != nil {
		group.Name = (*tag)[1]
	}
	if tag := evt.Tags.GetFirst([]string{"about", ""}); tag != nil {
		group.About = (*tag)[1]
	}
	if tag := evt.Tags.GetFirst([]string{"picture", ""}); tag != nil {
		group.Picture = (*tag)[1]
	}

	if tag := evt.Tags.GetFirst([]string{"private"}); tag != nil {
		group.Private = true
	}
	if tag := evt.Tags.GetFirst([]string{"closed"}); tag != nil {
		group.Closed = true
	}

	return nil
}
