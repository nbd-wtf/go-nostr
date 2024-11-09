package nip29

import (
	"testing"

	"github.com/stretchr/testify/require"
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

	require.Equal(t, "xyz", meta1.Tags.GetD(), "translation of group1 to metadata event failed: %s", meta1)
	require.NotNil(t, meta1.Tags.GetFirst([]string{"name", "banana"}), "translation of group1 to metadata event failed: %s", meta1)
	require.NotNil(t, meta1.Tags.GetFirst([]string{"private"}), "translation of group1 to metadata event failed: %s", meta1)

	group2, _ := NewGroup("groups.com'abc")
	group2.Members[ALICE] = []*Role{{Name: "nada"}}
	group2.Members[BOB] = []*Role{{Name: "nada"}}
	group2.Members[CAROL] = nil
	group2.Members[DEREK] = nil
	admins2 := group2.ToAdminsEvent()

	require.Equal(t, "abc", admins2.Tags.GetD(), "translation of group2 to admins event failed")
	require.Equal(t, 3, len(admins2.Tags), "translation of group2 to admins event failed")
	require.NotNil(t, admins2.Tags.GetFirst([]string{"p", ALICE, "nada"}), "translation of group2 to admins event failed")
	require.NotNil(t, admins2.Tags.GetFirst([]string{"p", BOB, "nada"}), "translation of group2 to admins event failed")

	members2 := group2.ToMembersEvent()
	require.Equal(t, "abc", members2.Tags.GetD(), "translation of group2 to members2 event failed")
	require.Equal(t, 5, len(members2.Tags), "translation of group2 to members2 event failed")
	require.NotNil(t, members2.Tags.GetFirst([]string{"p", ALICE}), "translation of group2 to members2 event failed")
	require.NotNil(t, members2.Tags.GetFirst([]string{"p", BOB}), "translation of group2 to members2 event failed")
	require.NotNil(t, members2.Tags.GetFirst([]string{"p", CAROL}), "translation of group2 to members2 event failed")
	require.NotNil(t, members2.Tags.GetFirst([]string{"p", DEREK}), "translation of group2 to members2 event failed")

	group1.MergeInMembersEvent(members2)
	require.Equal(t, 4, len(group1.Members), "merge of members2 into group1 failed")
	require.Len(t, group1.Members[ALICE], 0, "merge of members2 into group1 failed")
	require.Len(t, group1.Members[DEREK], 0, "merge of members2 into group1 failed")

	group1.MergeInAdminsEvent(admins2)
	require.Equal(t, 4, len(group1.Members), "merge of admins2 into group1 failed")

	require.Equal(t, "nada", group1.Members[ALICE][0].Name, "merge of admins2 into group1 failed")
	require.Len(t, group1.Members[DEREK], 0, "merge of admins2 into group1 failed")

	group2.MergeInMetadataEvent(meta1)
	require.Equal(t, "banana", group2.Name, "merge of meta1 into group2 failed")
	require.Equal(t, "abc", group2.Address.ID, "merge of meta1 into group2 failed")
}
