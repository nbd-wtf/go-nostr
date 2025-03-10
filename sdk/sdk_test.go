package sdk

import (
	"context"
	"fmt"
	"strings"
	"sync"
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
	require.Contains(t, relays, "wss://lockbox.fiatjaf.com")
	require.Contains(t, relays, "wss://relay.westernbtc.com")

	// fetch notes
	filter := nostr.Filter{
		Kinds:   []int{1},
		Authors: []string{meta.PubKey},
		Limit:   5,
	}
	events := make([]*nostr.Event, 0, 5)
	for ie := range sys.Pool.FetchMany(ctx, relays, filter) {
		events = append(events, ie.Event)
	}
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(events), 5)
}

func TestConcurrentMetadata(t *testing.T) {
	sys := NewSystem()
	ctx := context.Background()

	wg := sync.WaitGroup{}
	for _, v := range []struct {
		input string
		name  string
	}{
		{
			"nprofile1qyxhwumn8ghj7mn0wvhxcmmvqyd8wumn8ghj7un9d3shjtnhv4ehgetjde38gcewvdhk6qpq80cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwswpnfsn",
			"fiatjaf",
		},
		{
			"npub1t6jxfqz9hv0lygn9thwndekuahwyxkgvycyscjrtauuw73gd5k7sqvksrw",
			"constant",
		},
		{
			"npub1jlrs53pkdfjnts29kveljul2sm0actt6n8dxrrzqcersttvcuv3qdjynqn",
			"hodlbod",
		},
		{
			"npub1xtscya34g58tk0z605fvr788k263gsu6cy9x0mhnm87echrgufzsevkk5s",
			"jb55",
		},
		{
			"npub1qny3tkh0acurzla8x3zy4nhrjz5zd8l9sy9jys09umwng00manysew95gx",
			"odell",
		},
		{
			"npub1l2vyh47mk2p0qlsku7hg0vn29faehy9hy34ygaclpn66ukqp3afqutajft",
			"pablo",
		},
	} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			meta, err := sys.FetchProfileFromInput(ctx, v.input)
			require.NoError(t, err)
			require.Contains(t, strings.ToLower(meta.Name), v.name)

			fl := sys.FetchFollowList(ctx, meta.PubKey)
			require.GreaterOrEqual(t, len(fl.Items), 30, "%s/%s", meta.PubKey, meta.Name)
		}()
	}

	wg.Wait()
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
	wg := sync.WaitGroup{}
	for i, item := range followList.Items[0:120] {
		wg.Add(1)
		go func() {
			meta := sys.FetchProfileMetadata(ctx, item.Pubkey)
			fl := sys.FetchFollowList(ctx, item.Pubkey)
			fmt.Println("  ~", i, item.Pubkey, len(fl.Items))
			results <- result{item.Pubkey, fl, meta}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// collect results
	var validAccounts int
	var accountsWithManyFollows int
	for r := range results {
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
	require.Greater(t, ratio, 0.7, "at least 70%% of accounts should follow more than 20 others (actual: %.2f%%)", ratio*100)
}
