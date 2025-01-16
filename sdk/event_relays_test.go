package sdk

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeRelayList(t *testing.T) {
	tests := []struct {
		name   string
		relays []string
	}{
		{
			name:   "empty list",
			relays: []string{},
		},
		{
			name:   "single relay",
			relays: []string{"wss://relay.example.com"},
		},
		{
			name: "multiple relays",
			relays: []string{
				"wss://relay1.example.com",
				"wss://relay23.example.com",
				"wss://relay456.example.com",
			},
		},
		{
			name: "relays with varying lengths",
			relays: []string{
				"wss://a.com",
				"wss://very-long-relay-url.example.com",
				"wss://b.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// test encoding
			encoded := encodeRelayList(tt.relays)
			require.NotNil(t, encoded)

			// test decoding
			decoded := decodeRelayList(encoded)
			require.Equal(t, tt.relays, decoded)
		})
	}

	t.Run("malformed data", func(t *testing.T) {
		// test with truncated data
		decoded := decodeRelayList([]byte{5, 'h', 'e'}) // length prefix of 5 but only 2 bytes of data
		require.Nil(t, decoded)

		// test with invalid length prefix
		decoded = decodeRelayList([]byte{255}) // length prefix but no data
		require.Nil(t, decoded)
	})

	t.Run("skip too long relay URLs", func(t *testing.T) {
		// create a long URL by repeating 'a' 257 times
		longURL := "wss://" + strings.Repeat("a", 257) + ".com"
		longRelays := []string{
			"wss://normal.example.com",
			longURL,
			"wss://also-normal.example.com",
		}

		encoded := encodeRelayList(longRelays)
		decoded := decodeRelayList(encoded)

		// should only contain the normal URLs
		require.Equal(t, []string{
			"wss://normal.example.com",
			"wss://also-normal.example.com",
		}, decoded)
	})
}
