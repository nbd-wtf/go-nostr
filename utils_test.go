package nostr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidRelayURL(t *testing.T) {
	tests := []struct {
		u    string
		want bool
	}{
		{"ws://127.0.0.1", true},
		{"ws://localhost", true},
		{"wss://localhost", true},
		{"wss://relay.nostr.com", true},
		{"http://127.0.0.1", false},
		{"127.0.0.1", false},
	}

	for _, test := range tests {
		got := IsValidRelayURL(test.u)
		assert.Equal(t, test.want, got)
	}
}
