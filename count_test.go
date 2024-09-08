package nostr

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCount(t *testing.T) {
	const RELAY = "wss://relay.nostr.band"

	rl := mustRelayConnect(t, RELAY)
	defer rl.Close()

	count, err := rl.Count(context.Background(), Filters{
		{Kinds: []int{KindContactList}, Tags: TagMap{"p": []string{"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"}}},
	})
	assert.NoError(t, err)
	assert.Greater(t, count, int64(0))
}
