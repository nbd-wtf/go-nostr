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
}

func (group Group) MetadataEvent() *nostr.Event {
	evt := &nostr.Event{
		Kind:      39000,
		CreatedAt: nostr.Now(),
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

func MetadataEventToGroup(evt *nostr.Event) (*Group, error) {
	if evt.Kind != nostr.KindSimpleGroupMetadata {
		return nil, fmt.Errorf("expected kind %d, got %d", nostr.KindSimpleGroupMetadata, evt.Kind)
	}

	group := &Group{}
	group.ID = evt.Tags.GetD()

    if tag := nostr.Tags.GetFirst([]string{"name", ""}); tag != nil {
    group.Name = (*tag)[1]
    }
    if tag := nostr.Tags.GetFirst([]string{"about", ""}); tag != nil {
    group.About = (*tag)[1]
    }
    if tag := nostr.Tags.GetFirst([]string{"picture", ""}); tag != nil {
    group.Picture = (*tag)[1]
    }


	Members map[string]*Role
	Private bool
	Closed  bool
}
