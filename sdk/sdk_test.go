package sdk

import (
	"context"
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestSystemFiatjaf(t *testing.T) {
	sys := NewSystem()
	ctx := context.Background()

	// get metadata
	meta, err := sys.FetchProfileFromInput(ctx, "nprofile1qyxhwumn8ghj7mn0wvhxcmmvqyd8wumn8ghj7un9d3shjtnhv4ehgetjde38gcewvdhk6qpq80cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwswpnfsn")
	require.NoError(t, err)
	require.Equal(t, "fiatjaf", meta.Name)

	// check outbox relays
	relays := sys.FetchOutboxRelays(ctx, meta.PubKey, 5)
	require.Contains(t, relays, "wss://relay.westernbtc.com")
	require.Contains(t, relays, "wss://pyramid.fiatjaf.com")

	// fetch notes
	filter := nostr.Filter{
		Kinds:   []int{1},
		Authors: []string{meta.PubKey},
		Limit:   5,
	}
	events, err := sys.FetchUserEvents(ctx, filter)
	require.NoError(t, err)
	require.NotEmpty(t, events[meta.PubKey])
	require.GreaterOrEqual(t, len(events[meta.PubKey]), 5)
}
