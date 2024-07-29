package nostr

import (
	"testing"
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
		//{"wss://relay.nostr.com'", false},
		//{"wss://relay.nostr.com'hiphop", true},
	}

	for _, test := range tests {
		got := IsValidRelayURL(test.u)
		if got != test.want {
			t.Errorf("IsValidRelayURL want %v for %q but got %v", test.want, test.u, got)
		}
	}
}
