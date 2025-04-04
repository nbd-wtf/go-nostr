package sdk

import (
	"sync"
	"testing"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestLoadWoT(t *testing.T) {
	sys := NewSystem()
	ctx := t.Context()

	// test with fiatjaf's pubkey
	wotch, err := sys.loadWoT(ctx, "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d")
	require.NoError(t, err)

	wot := make([]string, 0, 100000)
	wotch2 := make(chan string)

	var filter WotXorFilter
	done := make(chan struct{})
	go func() {
		// test that we can get a filter from the WoT
		filter = makeWoTFilter(wotch2)
		close(done)
	}()

	for pk := range wotch {
		wot = append(wot, pk)
		wotch2 <- pk
	}
	close(wotch2)

	// we should get a decent number of pubkeys in the WoT
	require.Greater(t, len(wot), 10000, "should have more than 10000 pubkeys in WoT")

	// test that the filter contains some known pubkeys from the WoT
	<-done
	for _, pk := range wot {
		require.True(t, filter.Contains(pk), "filter should contain all WoT pubkeys")
	}
}

func TestLoadWoTManyPeople(t *testing.T) {
	sys := NewSystem()
	ctx := t.Context()

	wg := sync.WaitGroup{}
	wg.Add(3 + 2 + 2)

	diffs := make([]nostr.Timestamp, 5)
	var rabble1 WotXorFilter
	var rabble2 WotXorFilter
	var rabble3 WotXorFilter
	var alex1 WotXorFilter
	var alex2 WotXorFilter

	// these are the same pubkey
	go func() {
		rabble, err := sys.LoadWoTFilter(ctx, "76c71aae3a491f1d9eec47cba17e229cda4113a0bbb6e6ae1776d7643e29cafa")
		require.NoError(t, err)
		diffs[0] = nostr.Now()
		rabble1 = rabble
		wg.Done()
	}()

	time.Sleep(time.Millisecond * 20)
	go func() {
		rabble, err := sys.LoadWoTFilter(ctx, "76c71aae3a491f1d9eec47cba17e229cda4113a0bbb6e6ae1776d7643e29cafa")
		require.NoError(t, err)
		diffs[1] = nostr.Now()
		rabble2 = rabble
		wg.Done()
	}()

	time.Sleep(time.Millisecond * 20)
	go func() {
		rabble, err := sys.LoadWoTFilter(ctx, "76c71aae3a491f1d9eec47cba17e229cda4113a0bbb6e6ae1776d7643e29cafa")
		require.NoError(t, err)
		diffs[2] = nostr.Now()
		rabble3 = rabble
		wg.Done()
	}()

	// these should map to the same pos
	time.Sleep(time.Millisecond * 20)
	go func() {
		alex, err := sys.LoadWoTFilter(ctx, "9ce71f1506ccf4b99f234af49bd6202be883a80f95a155c6e9a1c36fd7e780c7")
		require.NoError(t, err)
		diffs[3] = nostr.Now()
		alex1 = alex
		wg.Done()
	}()

	time.Sleep(time.Millisecond * 20)
	go func() {
		alex, err := sys.LoadWoTFilter(ctx, "9ce71f1506ccf4b99f234af49bd6202be883a80f95a155c6e9a1c36fd7e780c7")
		require.NoError(t, err)
		diffs[4] = nostr.Now()
		alex2 = alex
		wg.Done()
	}()

	// these are independent
	go func() {
		hodlbod, err := sys.LoadWoTFilter(ctx, "97c70a44366a6535c145b333f973ea86dfdc2d7a99da618c40c64705ad98e322")
		require.NoError(t, err)
		require.True(t, hodlbod.Contains("ee11a5dff40c19a555f41fe42b48f00e618c91225622ae37b6c2bb67b76c4e49"))
		require.True(t, hodlbod.Contains("76c71aae3a491f1d9eec47cba17e229cda4113a0bbb6e6ae1776d7643e29cafa"))
		require.True(t, hodlbod.Contains("3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"))
		wg.Done()
	}()
	go func() {
		mikedilger, err := sys.LoadWoTFilter(ctx, "ee11a5dff40c19a555f41fe42b48f00e618c91225622ae37b6c2bb67b76c4e49")
		require.NoError(t, err)
		require.True(t, mikedilger.Contains("97c70a44366a6535c145b333f973ea86dfdc2d7a99da618c40c64705ad98e322"))
		require.True(t, mikedilger.Contains("3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"))
		wg.Done()
	}()

	wg.Wait()

	require.Equal(t, rabble1, rabble2)
	require.Equal(t, rabble2, rabble3)
	require.Equal(t, alex1, alex2)

	require.Less(t, int(diffs[1]-diffs[0]), 1, "second duplicated call should resolve immediately")
	require.Less(t, int(diffs[2]-diffs[1]), 1, "third duplicated call should resolve immediately")
	require.Greater(t, int(diffs[3]-diffs[2]), 10, "the next call should take a long time")
	require.Less(t, int(diffs[4]-diffs[3]), 1, "and then a duplicated call should resolve immediately")
}
