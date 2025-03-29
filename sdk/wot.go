package sdk

import (
	"context"
	"maps"
	"slices"
	"strconv"

	"github.com/FastFilter/xorfilter"
	"golang.org/x/sync/errgroup"
)

func (sys *System) GetWoT(ctx context.Context, pubkey string) (map[uint64]struct{}, error) {
	g, ctx := errgroup.WithContext(ctx)

	res := make(chan uint64)
	for _, f := range sys.FetchFollowList(ctx, pubkey).Items {
		g.Go(func() error {
			for _, f2 := range sys.FetchFollowList(ctx, f.Pubkey).Items {
				shid, _ := strconv.ParseUint(f2.Pubkey[32:48], 16, 64)
				res <- shid
			}
			return nil
		})
	}

	result := make(map[uint64]struct{})
	go func() {
		for shid := range res {
			result[shid] = struct{}{}
		}
	}()

	return result, g.Wait()
}

func (sys *System) GetWoTFilter(ctx context.Context, pubkey string) (*xorfilter.Xor8, error) {
	m, err := sys.GetWoT(ctx, pubkey)
	if err != nil {
		return nil, err
	}

	return xorfilter.Populate(slices.Collect(maps.Keys(m)))
}
