package sdk

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/sdk/dataloader"
)

type replaceableIndex int

const (
	kind_0     replaceableIndex = 0
	kind_3     replaceableIndex = 1
	kind_10000 replaceableIndex = 2
	kind_10001 replaceableIndex = 3
	kind_10002 replaceableIndex = 4
	kind_10003 replaceableIndex = 5
	kind_10004 replaceableIndex = 6
	kind_10005 replaceableIndex = 7
	kind_10006 replaceableIndex = 8
	kind_10007 replaceableIndex = 9
	kind_10015 replaceableIndex = 10
	kind_10030 replaceableIndex = 11
)

type EventResult dataloader.Result[*nostr.Event]

func (sys *System) initializeReplaceableDataloaders() {
	sys.replaceableLoaders = make([]*dataloader.Loader[string, *nostr.Event], 12)
	sys.replaceableLoaders[kind_0] = sys.createReplaceableDataloader(0)
	sys.replaceableLoaders[kind_3] = sys.createReplaceableDataloader(3)
	sys.replaceableLoaders[kind_10000] = sys.createReplaceableDataloader(10000)
	sys.replaceableLoaders[kind_10001] = sys.createReplaceableDataloader(10001)
	sys.replaceableLoaders[kind_10002] = sys.createReplaceableDataloader(10002)
	sys.replaceableLoaders[kind_10003] = sys.createReplaceableDataloader(10003)
	sys.replaceableLoaders[kind_10004] = sys.createReplaceableDataloader(10004)
	sys.replaceableLoaders[kind_10005] = sys.createReplaceableDataloader(10005)
	sys.replaceableLoaders[kind_10006] = sys.createReplaceableDataloader(10006)
	sys.replaceableLoaders[kind_10007] = sys.createReplaceableDataloader(10007)
	sys.replaceableLoaders[kind_10015] = sys.createReplaceableDataloader(10015)
	sys.replaceableLoaders[kind_10030] = sys.createReplaceableDataloader(10030)
}

func (sys *System) createReplaceableDataloader(kind int) *dataloader.Loader[string, *nostr.Event] {
	return dataloader.NewBatchedLoader(
		func(ctxs []context.Context, pubkeys []string) map[string]dataloader.Result[*nostr.Event] {
			return sys.batchLoadReplaceableEvents(ctxs, kind, pubkeys)
		},
		dataloader.Options{
			Wait:         time.Millisecond * 110,
			MaxThreshold: 30,
		},
	)
}

func (sys *System) batchLoadReplaceableEvents(
	ctxs []context.Context,
	kind int,
	pubkeys []string,
) map[string]dataloader.Result[*nostr.Event] {
	batchSize := len(pubkeys)
	results := make(map[string]dataloader.Result[*nostr.Event], batchSize)
	relayFilter := make([]nostr.DirectedFilter, 0, max(3, batchSize*2))
	relayFilterIndex := make(map[string]int, max(3, batchSize*2))

	wg := sync.WaitGroup{}
	wg.Add(len(pubkeys))
	cm := sync.Mutex{}

	aggregatedContext, aggregatedCancel := context.WithCancel(context.Background())
	waiting := atomic.Int32{}
	waiting.Add(int32(len(pubkeys)))

	for i, pubkey := range pubkeys {
		ctx, cancel := context.WithCancel(ctxs[i])
		defer cancel()

		// build batched queries for the external relays
		go func(i int, pubkey string) {
			// gather relays we'll use for this pubkey
			relays := sys.determineRelaysToQuery(ctx, pubkey, kind)

			cm.Lock()
			for _, relay := range relays {
				// each relay will have a custom filter
				idx, ok := relayFilterIndex[relay]
				var dfilter nostr.DirectedFilter
				if ok {
					dfilter = relayFilter[idx]
				} else {
					dfilter = nostr.DirectedFilter{
						Relay: relay,
						Filter: nostr.Filter{
							Kinds:   []int{kind},
							Authors: make([]string, 0, batchSize-i /* this and all pubkeys after this can be added */),
						},
					}
					idx = len(relayFilter)
					relayFilterIndex[relay] = idx
					relayFilter = append(relayFilter, dfilter)
				}
				dfilter.Authors = append(dfilter.Authors, pubkey)
				relayFilter[idx] = dfilter
			}
			cm.Unlock()
			wg.Done()

			<-ctx.Done()
			if waiting.Add(-1) == 0 {
				aggregatedCancel()
			}
		}(i, pubkey)
	}

	// query all relays with the prepared filters
	wg.Wait()
	multiSubs := sys.Pool.BatchedSubManyEose(aggregatedContext, relayFilter,
		nostr.WithLabel("repl~"+strconv.Itoa(kind)),
	)
	for {
		select {
		case ie, more := <-multiSubs:
			if !more {
				return results
			}

			// insert this event at the desired position
			if val, ok := results[ie.PubKey]; !ok || val.Data == nil || val.Data.CreatedAt < ie.CreatedAt {
				results[ie.PubKey] = dataloader.Result[*nostr.Event]{Data: ie.Event}
			}
		case <-aggregatedContext.Done():
			return results
		}
	}
}

func (sys *System) determineRelaysToQuery(ctx context.Context, pubkey string, kind int) []string {
	var relays []string

	// search in specific relays for user
	if kind == 10002 {
		// prevent infinite loops by jumping directly to this
		relays = sys.Hints.TopN(pubkey, 3)
	} else {
		ctx, cancel := context.WithTimeoutCause(ctx, time.Millisecond*2300,
			errors.New("fetching relays in subloader took too long"),
		)

		if kind == 0 || kind == 3 {
			// leave room for two hardcoded relays because people are stupid
			relays = sys.FetchOutboxRelays(ctx, pubkey, 1)
		} else {
			relays = sys.FetchOutboxRelays(ctx, pubkey, 3)
		}

		cancel()
	}

	// use a different set of extra relays depending on the kind
	needed := 3 - len(relays)
	for range needed {
		var next string
		switch kind {
		case 0:
			next = sys.MetadataRelays.Next()
		case 3:
			next = sys.FollowListRelays.Next()
		case 10002:
			next = sys.RelayListRelays.Next()
		default:
			next = sys.FallbackRelays.Next()
		}

		if !slices.Contains(relays, next) {
			relays = append(relays, next)
		}
	}

	return relays
}
