package nip29

import (
	"testing"
)

const (
	ALICE = "eadad094b75b4690e7ee7124522861b8d81d5ed92e81eb678e776d1164d1efe9"
	BOB   = "6ac475cdf30e2006ee5142559544e86f8f1b485a9c8c1f2da467996fb7fcdfe7"
	CAROL = "f81982b8b6ba354a1e09acfda348512ef93e5778847fb5f4b30fe6b0042f4b36"
	DEREK = "24a049c4e5c9cff1764c312b2e0fa59a02af235b37809180b3f2c7b2ec3dbdfd"
)

func TestGroupEventBackAndForth(t *testing.T) {
	group1 := NewGroup("xyz")
	group1.Name = "banana"
	group1.Private = true
	meta1 := group1.ToMetadataEvent()
	if meta1.Tags.GetD() != "xyz" ||
		meta1.Tags.GetFirst([]string{"name", "banana"}) == nil ||
		meta1.Tags.GetFirst([]string{"private"}) == nil {
		t.Fatalf("translation of group1 to meta1data event failed")
	}

	group2 := NewGroup("abc")
	group2.Members[ALICE] = &Role{Name: "nada", Permissions: map[Permission]struct{}{PermAddUser: {}}}
	group2.Members[BOB] = &Role{Name: "nada", Permissions: map[Permission]struct{}{PermEditMetadata: {}}}
	group2.Members[CAROL] = EmptyRole
	group2.Members[DEREK] = EmptyRole
	admins2 := group2.ToAdminsEvent()
	if admins2.Tags.GetD() != "abc" ||
		len(admins2.Tags) != 3 ||
		admins2.Tags.GetFirst([]string{"p", ALICE, "nada", "add-user"}) == nil ||
		admins2.Tags.GetFirst([]string{"p", BOB, "nada", "edit-metadata"}) == nil {
		t.Fatalf("translation of group2 to admins event failed")
	}

	members2 := group2.ToMembersEvent()
	if members2.Tags.GetD() != "abc" ||
		len(members2.Tags) != 5 ||
		members2.Tags.GetFirst([]string{"p", ALICE}) == nil ||
		members2.Tags.GetFirst([]string{"p", BOB}) == nil ||
		members2.Tags.GetFirst([]string{"p", CAROL}) == nil ||
		members2.Tags.GetFirst([]string{"p", DEREK}) == nil {
		t.Fatalf("translation of group2 to members2 event failed")
	}

	group1.MergeInMembersEvent(members2)
	if len(group1.Members) != 4 || group1.Members[ALICE] != EmptyRole || group1.Members[DEREK] != EmptyRole {
		t.Fatalf("merge of members2 into group1 failed")
	}
	group1.MergeInAdminsEvent(admins2)
	if len(group1.Members) != 4 || group1.Members[ALICE].Name != "nada" || group1.Members[DEREK] != EmptyRole {
		t.Fatalf("merge of admins2 into group1 failed")
	}

	group2.MergeInMetadataEvent(meta1)
	if group2.Name != "banana" || group2.ID != "abc" {
		t.Fatalf("merge of meta1 into group2 failed")
	}
}
