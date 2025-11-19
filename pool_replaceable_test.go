package nostr

import (
	"testing"

	"github.com/puzpuzpuz/xsync/v3"
	"github.com/stretchr/testify/require"
)

func TestShouldDropReplaceable(t *testing.T) {
	seen := xsync.NewMapOf[ReplaceableKey, Timestamp]()
	key := ReplaceableKey{PubKey: "pk", D: "label"}

	require.False(t, shouldDropReplaceable(seen, key, Timestamp(1)))
	require.Equal(t, Timestamp(1), getReplaceableValue(t, seen, key))

	require.False(t, shouldDropReplaceable(seen, key, Timestamp(5)))
	require.Equal(t, Timestamp(5), getReplaceableValue(t, seen, key))

	require.True(t, shouldDropReplaceable(seen, key, Timestamp(3)))
	require.Equal(t, Timestamp(5), getReplaceableValue(t, seen, key))
}

func getReplaceableValue(t *testing.T, seen *xsync.MapOf[ReplaceableKey, Timestamp], key ReplaceableKey) Timestamp {
	value, ok := seen.Load(key)
	require.True(t, ok)
	return value
}
