package nostr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func KindKindTest(t *testing.T) {
	require.True(t, IsRegularKind(1))
	require.True(t, IsRegularKind(9))
	require.True(t, IsRegularKind(1111))
	require.True(t, IsReplaceableKind(0))
	require.True(t, IsReplaceableKind(3))
	require.True(t, IsReplaceableKind(10002))
	require.True(t, IsReplaceableKind(10050))
	require.True(t, IsAddressableKind(30023))
	require.True(t, IsAddressableKind(39000))
}
