package nostr

import (
	"context"
	"testing"
)

func TestCount(t *testing.T) {
	const RELAY = "wss://relay.nostr.band"

	rl := mustRelayConnect(RELAY)
	defer rl.Close()

	count, err := rl.Count(context.Background(), Filters{
		{Kinds: []int{3}, Tags: TagMap{"p": []string{"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"}}},
	})
	if err != nil {
		t.Errorf("count request failed: %v", err)
		return
	}

	if count <= 0 {
		t.Errorf("count result wrong: %v", count)
		return
	}
}
