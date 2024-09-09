package nip29

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	ALICE = "eadad094b75b4690e7ee7124522861b8d81d5ed92e81eb678e776d1164d1efe9"
	BOB   = "6ac475cdf30e2006ee5142559544e86f8f1b485a9c8c1f2da467996fb7fcdfe7"
	CAROL = "f81982b8b6ba354a1e09acfda348512ef93e5778847fb5f4b30fe6b0042f4b36"
	DEREK = "24a049c4e5c9cff1764c312b2e0fa59a02af235b37809180b3f2c7b2ec3dbdfd"
)

func TestGroupEventBackAndForth(t *testing.T) {
	group1, _ := NewGroup("relay.com'xyz")
	group1.Name = "banana"
	group1.Private = true
	meta1 := group1.ToMetadataEvent()

	assert.Equal(t, "xyz", meta1.Tags.GetD(), "translation of group1 to metadata event failed: %s", meta1)
	assert.NotNil(t, meta1.Tags.GetFirst([]string{"name", "banana"}), "translation of group1 to metadata event failed: %s", meta1)
	assert.NotNil(t, meta1.Tags.GetFirst([]string{"private"}), "translation of group1 to metadata event failed: %s", meta1)

	group2, _ := NewGroup("groups.com'abc")
	group2.Members[ALICE] = &Role{Name: "nada", Permissions: map[Permission]struct{}{PermAddUser: {}}}
	group2.Members[BOB] = &Role{Name: "nada", Permissions: map[Permission]struct{}{PermEditMetadata: {}}}
	group2.Members[CAROL] = EmptyRole
	group2.Members[DEREK] = EmptyRole
	admins2 := group2.ToAdminsEvent()

	assert.Equal(t, "abc", admins2.Tags.GetD(), "translation of group2 to admins event failed")
	assert.Equal(t, 3, len(admins2.Tags), "translation of group2 to admins event failed")
	assert.NotNil(t, admins2.Tags.GetFirst([]string{"p", ALICE, "nada", "add-user"}), "translation of group2 to admins event failed")
	assert.NotNil(t, admins2.Tags.GetFirst([]string{"p", BOB, "nada", "edit-metadata"}), "translation of group2 to admins event failed")

	members2 := group2.ToMembersEvent()
	assert.Equal(t, "abc", members2.Tags.GetD(), "translation of group2 to members2 event failed")
	assert.Equal(t, 5, len(members2.Tags), "translation of group2 to members2 event failed")
	assert.NotNil(t, members2.Tags.GetFirst([]string{"p", ALICE}), "translation of group2 to members2 event failed")
	assert.NotNil(t, members2.Tags.GetFirst([]string{"p", BOB}), "translation of group2 to members2 event failed")
	assert.NotNil(t, members2.Tags.GetFirst([]string{"p", CAROL}), "translation of group2 to members2 event failed")
	assert.NotNil(t, members2.Tags.GetFirst([]string{"p", DEREK}), "translation of group2 to members2 event failed")

	group1.MergeInMembersEvent(members2)
	assert.Equal(t, 4, len(group1.Members), "merge of members2 into group1 failed")
	assert.Equal(t, EmptyRole, group1.Members[ALICE], "merge of members2 into group1 failed")
	assert.Equal(t, EmptyRole, group1.Members[DEREK], "merge of members2 into group1 failed")

	group1.MergeInAdminsEvent(admins2)
	assert.Equal(t, 4, len(group1.Members), "merge of admins2 into group1 failed")
	assert.Equal(t, "nada", group1.Members[ALICE].Name, "merge of admins2 into group1 failed")
	assert.Equal(t, EmptyRole, group1.Members[DEREK], "merge of admins2 into group1 failed")

	group2.MergeInMetadataEvent(meta1)
	assert.Equal(t, "banana", group2.Name, "merge of meta1 into group2 failed")
	assert.Equal(t, "abc", group2.Address.ID, "merge of meta1 into group2 failed")
}
