package nostr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTagsClone(t *testing.T) {
	original := Tags{
		{"a", "1"},
		{"b", "2"},
	}

	clone := original.Clone()
	require.Equal(t, original, clone)

	clone = append(clone, Tag{"c", "3"})
	require.Len(t, original, 2)
}

func TestTagsCloneDeep(t *testing.T) {
	original := Tags{
		{"a", "1"},
		{"b", "2"},
	}

	clone := original.CloneDeep()
	require.Equal(t, original, clone)

	clone[0][1] = "updated"
	require.Equal(t, "1", original[0][1])
}

func TestTagClone(t *testing.T) {
	original := Tag{"e", "123", "relay"}
	clone := original.Clone()

	require.Equal(t, original, clone)

	clone[1] = "456"
	require.Equal(t, "123", original[1])
}
