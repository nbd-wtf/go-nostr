package nip02

import (
	"testing"
)

const RELAY = "wss://relay.nostr.band"

func TestGetContacts(t *testing.T) {

	t.Run("should return a list of contacts", func(t *testing.T) {
		npub := "npub19xcucl4ewfx40zfhdpsc6clqpl8yw340suzgyyee4mn5mpmhtt7qratgs9"

		got, err := GetContacts(npub, RELAY)
		if err != nil {
			t.Errorf("got unexpected error: %v", err)
		}

		if len(got) == 0 {
			t.Errorf("expected a non-empty slice, got: %v", got)
		}
	})
}
