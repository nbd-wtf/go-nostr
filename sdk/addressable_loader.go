package sdk

import (
	"context"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/sdk/dataloader"
)

// this is similar to replaceable_loader and reuses logic from that.

type addressableIndex int

const (
	kind_30000 addressableIndex = 0
	kind_30002 addressableIndex = 1
	kind_30015 addressableIndex = 2
	kind_30030 addressableIndex = 3
)

func (sys *System) initializeAddressableDataloaders() {
	sys.addressableLoaders = make([]*dataloader.Loader[string, []*nostr.Event], 4)
	sys.addressableLoaders[kind_30000] = sys.createAddressableDataloader(30000)
	sys.addressableLoaders[kind_30002] = sys.createAddressableDataloader(30002)
	sys.addressableLoaders[kind_30015] = sys.createAddressableDataloader(30015)
	sys.addressableLoaders[kind_30030] = sys.createAddressableDataloader(30030)
}

func (sys *System) createAddressableDataloader(kind int) *dataloader.Loader[string, []*nostr.Event] {
	return dataloader.NewBatchedLoader(
		func(ctxs []context.Context, pubkeys []string) map[string]dataloader.Result[[]*nostr.Event] {
			return sys.batchLoadAddressableEvents(ctxs, kind, pubkeys)
		},
		dataloader.Options{
			Wait:         time.Millisecond * 110,
			MaxThreshold: 30,
		},
	)
}

func (sys *System) batchLoadAddressableEvents(
	ctxs []context.Context,
	kind int,
	pubkeys []string,
) map[string]dataloader.Result[[]*nostr.Event] {
	batchSize := len(pubkeys)
	results := make(map[string]dataloader.Result[[]*nostr.Event], batchSize)
	relayFilter := make([]nostr.DirectedFilter, 0, max(3, batchSize*2))
	relayFilterIndex := make(map[string]int, max(3, batchSize*2))

	wg := sync.WaitGroup{}
	wg.Add(len(pubkeys))
	cm := sync.Mutex{}

	aggregatedContext, aggregatedCancel := context.WithCancel(context.Background())
	waiting := len(pubkeys)

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
			waiting--
			if waiting == 0 {
				aggregatedCancel()
			}
		}(i, pubkey)
	}

	// wait for relay batches to be prepared
	wg.Wait()

	// query all relays with the prepared filters
	multiSubs := sys.Pool.BatchedSubManyEose(aggregatedContext, relayFilter)
nextEvent:
	for {
		select {
		case ie, more := <-multiSubs:
			if !more {
				return results
			}

			events := results[ie.PubKey].Data
			if events == nil {
				// no events found, so just add this and end
				results[ie.PubKey] = dataloader.Result[[]*nostr.Event]{Data: []*nostr.Event{ie.Event}}
				continue nextEvent
			}

			// there are events, so look for a match
			d := ie.Tags.GetD()
			for i, event := range events {
				if event.Tags.GetD() == d {
					// there is a match
					if event.CreatedAt < ie.CreatedAt {
						// ...and this one is newer, so replace
						events[i] = ie.Event
					} else {
						// ... but this one is older, so ignore
					}
					// in any case we end this here
					continue nextEvent
				}
			}

			events = append(events, ie.Event)
			results[ie.PubKey] = dataloader.Result[[]*nostr.Event]{Data: events}
		case <-aggregatedContext.Done():
			return results
		}
	}
}
