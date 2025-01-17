package sdk

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/graph-gophers/dataloader/v7"
	"github.com/nbd-wtf/go-nostr"
)

// this is used as a hack to signal that these replaceable loader queries shouldn't use the full
// context timespan when they're being made from inside determineRelaysToQuery
var contextForSub10002Query = context.WithValue(context.Background(), "", "")

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
		func(ctx context.Context, pubkeys []string) []*dataloader.Result[*nostr.Event] {
			var cancel context.CancelFunc

			if ctx == contextForSub10002Query {
				ctx, cancel = context.WithTimeoutCause(context.Background(), time.Millisecond*2300,
					errors.New("fetching relays in subloader took too long"),
				)
			} else {
				ctx, cancel = context.WithTimeoutCause(context.Background(), time.Second*6,
					errors.New("batch replaceable load took too long"),
				)
				defer cancel()
			}

			return sys.batchLoadReplaceableEvents(ctx, kind, pubkeys)
		},
		dataloader.WithBatchCapacity[string, *nostr.Event](60),
		dataloader.WithClearCacheOnBatch[string, *nostr.Event](),
		dataloader.WithCache(&dataloader.NoCache[string, *nostr.Event]{}),
		dataloader.WithWait[string, *nostr.Event](time.Millisecond*350),
	)
}

func (sys *System) batchLoadReplaceableEvents(
	ctx context.Context,
	kind int,
	pubkeys []string,
) []*dataloader.Result[*nostr.Event] {
	batchSize := len(pubkeys)
	results := make([]*dataloader.Result[*nostr.Event], batchSize)
	keyPositions := make(map[string]int) // { [pubkey]: slice_index }
	relayFilter := make([]nostr.DirectedFilter, 0, max(3, batchSize*2))
	relayFilterIndex := make(map[string]int, max(3, batchSize*2))

	wg := sync.WaitGroup{}
	wg.Add(len(pubkeys))
	cm := sync.Mutex{}

	for i, pubkey := range pubkeys {
		// build batched queries for the external relays
		keyPositions[pubkey] = i // this is to help us know where to save the result later

		go func(i int, pubkey string) {
			defer wg.Done()

			// if we're attempting this query with a short key (last 8 characters), stop here
			if len(pubkey) != 64 {
				results[i] = &dataloader.Result[*nostr.Event]{
					Error: fmt.Errorf("won't proceed to query relays with a shortened key (%d)", kind),
				}
				return
			}

			// save attempts here so we don't try the same failed query over and over
			if doItNow := doThisNotMoreThanOnceAnHour("repl:" + strconv.Itoa(kind) + pubkey); !doItNow {
				results[i] = &dataloader.Result[*nostr.Event]{
					Error: fmt.Errorf("last attempt failed, waiting more to try again"),
				}
				return
			}

			// gather relays we'll use for this pubkey
			relays := sys.determineRelaysToQuery(ctx, pubkey, kind)

			// by default we will return an error (this will be overwritten when we find an event)
			results[i] = &dataloader.Result[*nostr.Event]{
				Error: fmt.Errorf("couldn't find a kind %d event anywhere %v", kind, relays),
			}

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
		}(i, pubkey)
	}

	// query all relays with the prepared filters
	wg.Wait()
	multiSubs := sys.Pool.BatchedSubManyEose(ctx, relayFilter, nostr.WithLabel("repl~"+strconv.Itoa(kind)))
	for {
		select {
		case ie, more := <-multiSubs:
			if !more {
				return results
			}

			// insert this event at the desired position
			pos := keyPositions[ie.PubKey] // @unchecked: it must succeed because it must be a key we passed
			if results[pos].Data == nil || results[pos].Data.CreatedAt < ie.CreatedAt {
				results[pos] = &dataloader.Result[*nostr.Event]{Data: ie.Event}
			}
		case <-ctx.Done():
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
		if len(relays) == 0 {
			relays = []string{"wss://relay.damus.io", "wss://nos.lol"}
		}
	} else {
		if kind == 0 || kind == 3 {
			// leave room for two hardcoded relays because people are stupid
			relays = sys.FetchOutboxRelays(contextForSub10002Query, pubkey, 1)
		} else {
			relays = sys.FetchOutboxRelays(contextForSub10002Query, pubkey, 3)
		}
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
