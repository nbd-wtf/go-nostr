package nip29

import (
	"slices"

	"github.com/nbd-wtf/go-nostr"
)

type Role struct {
	Name        string
	Description string
}

type KindRange []int

var ModerationEventKinds = KindRange{
	nostr.KindSimpleGroupPutUser,
	nostr.KindSimpleGroupRemoveUser,
	nostr.KindSimpleGroupEditMetadata,
	nostr.KindSimpleGroupDeleteEvent,
	nostr.KindSimpleGroupCreateGroup,
	nostr.KindSimpleGroupDeleteGroup,
	nostr.KindSimpleGroupCreateInvite,
}

var MetadataEventKinds = KindRange{
	nostr.KindSimpleGroupMetadata,
	nostr.KindSimpleGroupAdmins,
	nostr.KindSimpleGroupMembers,
	nostr.KindSimpleGroupRoles,
}

func (kr KindRange) Includes(kind int) bool {
	_, ok := slices.BinarySearch(kr, kind)
	return ok
}
