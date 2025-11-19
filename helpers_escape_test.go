package nostr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEscapeStringControlCharacters(t *testing.T) {
	raw := "\x00\x01\b\t\n\x1f"
	got := string(escapeString(nil, raw))
	require.Equal(t, `"\u0000\u0001\b\t\n\u001f"`, got)
}

func TestExtractEventFieldsOutOfBounds(t *testing.T) {
	malformed := `["EVENT","sub",{"id":"abc","pubkey":"dead"}]`

	require.Equal(t, "", extractEventID(malformed))
	require.Equal(t, "", extractEventPubKey(malformed))
}
