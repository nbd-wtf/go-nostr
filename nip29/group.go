package nip29

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

type GroupAddress struct {
	Relay string
	ID    string
}

func (gid GroupAddress) String() string {
	p, _ := url.Parse(gid.Relay)
	return fmt.Sprintf("%s'%s", p.Host, gid.ID)
}

func (gid GroupAddress) IsValid() bool {
	return gid.Relay != "" && gid.ID != ""
}

func (gid GroupAddress) Equals(gid2 GroupAddress) bool {
	return gid.Relay == gid2.Relay && gid.ID == gid2.ID
}

func ParseGroupAddress(raw string) (GroupAddress, error) {
	spl := strings.Split(raw, "'")
	if len(spl) != 2 {
		return GroupAddress{}, fmt.Errorf("invalid group id")
	}
	return GroupAddress{ID: spl[1], Relay: nostr.NormalizeURL(spl[0])}, nil
}

type Group struct {
	Address GroupAddress

	Name    string
	Picture string
	About   string
	Members map[string][]*Role
	Private bool
	Closed  bool

	Roles []*Role

	LastMetadataUpdate nostr.Timestamp
	LastAdminsUpdate   nostr.Timestamp
	LastMembersUpdate  nostr.Timestamp
	LastRolesUpdate    nostr.Timestamp
}

func (group Group) String() string {
	maybePrivate := ""
	maybeClosed := ""

	if group.Private {
		maybePrivate = " private"
	}
	if group.Closed {
		maybeClosed = " closed"
	}

	members := make([]string, len(group.Members))
	i := 0
	for pubkey, roles := range group.Members {
		members[i] = pubkey
		if len(roles) > 0 {
			members[i] += ":"
		}
		for _, role := range roles {
			members[i] += role.Name
			if slices.Contains(group.Roles, role) {
				members[i] += "*"
			}
			members[i] += "/"
		}
		members[i] = strings.TrimRight(members[i], "/")
		i++
	}

	return fmt.Sprintf(`<Group %s name="%s"%s%s picture="%s" about="%s" members=[%v]>`,
		group.Address,
		group.Name,
		maybePrivate,
		maybeClosed,
		group.Picture,
		group.About,
		strings.Join(members, " "),
	)
}

// NewGroup takes a group address in the form "<id>'<relay-hostname>"
func NewGroup(gadstr string) (Group, error) {
	gad, err := ParseGroupAddress(gadstr)
	if err != nil {
		return Group{}, fmt.Errorf("invalid group id '%s': %w", gadstr, err)
	}

	return Group{
		Address: gad,
		Name:    gad.ID,
		Members: make(map[string][]*Role),
	}, nil
}

func NewGroupFromMetadataEvent(relayURL string, evt *nostr.Event) (Group, error) {
	g := Group{
		Address: GroupAddress{
			Relay: relayURL,
			ID:    evt.Tags.GetD(),
		},
		Name:    evt.Tags.GetD(),
		Members: make(map[string][]*Role),
	}

	err := g.MergeInMetadataEvent(evt)
	return g, err
}

func (group Group) ToMetadataEvent() *nostr.Event {
	evt := &nostr.Event{
		Kind:      nostr.KindSimpleGroupMetadata,
		CreatedAt: group.LastMetadataUpdate,
		Tags: nostr.Tags{
			nostr.Tag{"d", group.Address.ID},
		},
	}
	if group.Name != "" {
		evt.Tags = append(evt.Tags, nostr.Tag{"name", group.Name})
	}
	if group.About != "" {
		evt.Tags = append(evt.Tags, nostr.Tag{"about", group.About})
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

func (group Group) ToAdminsEvent() *nostr.Event {
	evt := &nostr.Event{
		Kind:      nostr.KindSimpleGroupAdmins,
		CreatedAt: group.LastAdminsUpdate,
		Tags:      make(nostr.Tags, 1, 1+len(group.Members)/3),
	}
	evt.Tags[0] = nostr.Tag{"d", group.Address.ID}

	for member, roles := range group.Members {
		if len(roles) == 0 {
			// is not an admin
			continue
		}

		// is an admin
		tag := make([]string, 2, 2+len(roles))
		tag[0] = "p"
		tag[1] = member
		for _, role := range roles {
			tag = append(tag, role.Name)
		}
		evt.Tags = append(evt.Tags, tag)
	}

	return evt
}

func (group Group) ToMembersEvent() *nostr.Event {
	evt := &nostr.Event{
		Kind:      nostr.KindSimpleGroupMembers,
		CreatedAt: group.LastMembersUpdate,
		Tags:      make(nostr.Tags, 1, 1+len(group.Members)),
	}
	evt.Tags[0] = nostr.Tag{"d", group.Address.ID}

	for member := range group.Members {
		// include both admins and normal members
		evt.Tags = append(evt.Tags, nostr.Tag{"p", member})
	}

	return evt
}

func (group Group) ToRolesEvent() *nostr.Event {
	evt := &nostr.Event{
		Kind:      nostr.KindSimpleGroupRoles,
		CreatedAt: group.LastMembersUpdate,
		Tags:      make(nostr.Tags, 1, 1+len(group.Members)),
	}
	evt.Tags[0] = nostr.Tag{"d", group.Address.ID}

	for _, role := range group.Roles {
		// include both admins and normal members
		evt.Tags = append(evt.Tags, nostr.Tag{"role", role.Name, role.Description})
	}

	return evt
}

func (group *Group) MergeInMetadataEvent(evt *nostr.Event) error {
	if evt.Kind != nostr.KindSimpleGroupMetadata {
		return fmt.Errorf("expected kind %d, got %d", nostr.KindSimpleGroupMetadata, evt.Kind)
	}
	if evt.CreatedAt < group.LastMetadataUpdate {
		return fmt.Errorf("event is older than our last update (%d vs %d)", evt.CreatedAt, group.LastMetadataUpdate)
	}

	group.LastMetadataUpdate = evt.CreatedAt
	group.Name = group.Address.ID

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

func (group *Group) MergeInAdminsEvent(evt *nostr.Event) error {
	if evt.Kind != nostr.KindSimpleGroupAdmins {
		return fmt.Errorf("expected kind %d, got %d", nostr.KindSimpleGroupAdmins, evt.Kind)
	}
	if evt.CreatedAt < group.LastAdminsUpdate {
		return fmt.Errorf("event is older than our last update (%d vs %d)", evt.CreatedAt, group.LastAdminsUpdate)
	}

	group.LastAdminsUpdate = evt.CreatedAt
	for _, tag := range evt.Tags {
		if len(tag) < 3 {
			continue
		}
		if tag[0] != "p" {
			continue
		}
		if !nostr.IsValid32ByteHex(tag[1]) {
			continue
		}

		for _, roleName := range tag[2:] {
			group.Members[tag[1]] = append(group.Members[tag[1]], group.GetRoleByName(roleName))
		}
	}

	return nil
}

func (group *Group) MergeInMembersEvent(evt *nostr.Event) error {
	if evt.Kind != nostr.KindSimpleGroupMembers {
		return fmt.Errorf("expected kind %d, got %d", nostr.KindSimpleGroupMembers, evt.Kind)
	}
	if evt.CreatedAt < group.LastMembersUpdate {
		return fmt.Errorf("event is older than our last update (%d vs %d)", evt.CreatedAt, group.LastMembersUpdate)
	}

	group.LastMembersUpdate = evt.CreatedAt
	for _, tag := range evt.Tags {
		if len(tag) < 2 {
			continue
		}
		if tag[0] != "p" {
			continue
		}
		if !nostr.IsValid32ByteHex(tag[1]) {
			continue
		}

		_, exists := group.Members[tag[1]]
		if !exists {
			group.Members[tag[1]] = nil
		}
	}

	return nil
}
