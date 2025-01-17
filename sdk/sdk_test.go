package sdk

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestMetadataAndEvents(t *testing.T) {
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

func TestFollowListRecursion(t *testing.T) {
	sys := NewSystem()
	ctx := context.Background()

	// fetch initial follow list
	followList := sys.FetchFollowList(ctx, "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d")
	fmt.Println("~", len(followList.Items))
	require.Greater(t, len(followList.Items), 400, "should follow more than 400 accounts")

	// fetch metadata and follow lists for each followed account concurrently
	type result struct {
		pubkey     string
		followList GenericList[ProfileRef]
		metadata   ProfileMetadata
	}

	results := make(chan result)
	go func() {
		for _, item := range followList.Items {
			fl := sys.FetchFollowList(ctx, item.Pubkey)
			meta := sys.FetchProfileMetadata(ctx, item.Pubkey)
			fmt.Println("  ~", item.Pubkey, meta.Name, len(fl.Items))
			results <- result{item.Pubkey, fl, meta}
		}
	}()

	// collect results
	var validAccounts int
	var accountsWithManyFollows int
	for i := 0; i < len(followList.Items); i++ {
		r := <-results

		// skip if metadata has "bot" in name
		if strings.Contains(strings.ToLower(r.metadata.Name), "bot") {
			continue
		}

		validAccounts++
		if len(r.followList.Items) > 20 {
			accountsWithManyFollows++
		}
	}

	// check if at least 90% of non-bot accounts follow more than 20 accounts
	ratio := float64(accountsWithManyFollows) / float64(validAccounts)
	require.Greater(t, ratio, 0.9, "at least 90%% of accounts should follow more than 20 others (actual: %.2f%%)", ratio*100)
}
