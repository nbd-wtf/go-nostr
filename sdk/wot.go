package sdk

import (
	"context"
	"maps"
	"slices"
	"strconv"

	"github.com/FastFilter/xorfilter"
	"golang.org/x/sync/errgroup"
	"sync"
)

func PubKeyToShid(pubkey string) uint64 {
	shid, _ := strconv.ParseUint(pubkey[32:48], 16, 64)
	return shid
}

func (sys *System) GetWoT(ctx context.Context, pubkey string) (map[uint64]struct{}, error) {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(30)

	res := make(chan uint64, 100) // Add buffer to prevent blocking
	result := make(map[uint64]struct{})
	var resultMu sync.Mutex // Add mutex to protect map access

	// Start consumer goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		for shid := range res {
			resultMu.Lock()
			result[shid] = struct{}{}
			resultMu.Unlock()
		}
	}()

	// Process follow lists
	for _, f := range sys.FetchFollowList(ctx, pubkey).Items {
		f := f // Capture loop variable
		g.Go(func() error {
			for _, f2 := range sys.FetchFollowList(ctx, f.Pubkey).Items {
				select {
				case res <- PubKeyToShid(f2.Pubkey):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	}

	err := g.Wait()
	close(res) // Close channel after all goroutines are done
	<-done    // Wait for consumer to finish

	return result, err
}

func (sys *System) GetWoTFilter(ctx context.Context, pubkey string) (WotXorFilter, error) {
	m, err := sys.GetWoT(ctx, pubkey)
	if err != nil {
		return WotXorFilter{}, err
	}

	xf, err := xorfilter.Populate(slices.Collect(maps.Keys(m)))
	if err != nil {
		return WotXorFilter{}, err
	}

	return WotXorFilter{*xf}, nil
}

type WotXorFilter struct {
	xorfilter.Xor8
}

func (wxf WotXorFilter) Contains(pubkey string) bool {
	return wxf.Xor8.Contains(PubKeyToShid(pubkey))
}
