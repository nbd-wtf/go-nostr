package sdk

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/graph-gophers/dataloader/v7"
	"github.com/nbd-wtf/go-nostr"
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
		func(_ context.Context, pubkeys []string) []*dataloader.Result[[]*nostr.Event] {
			return sys.batchLoadAddressableEvents(kind, pubkeys)
		},
		dataloader.WithBatchCapacity[string, []*nostr.Event](30),
		dataloader.WithClearCacheOnBatch[string, []*nostr.Event](),
		dataloader.WithCache(&dataloader.NoCache[string, []*nostr.Event]{}),
		dataloader.WithWait[string, []*nostr.Event](time.Millisecond*350),
	)
}

func (sys *System) batchLoadAddressableEvents(
	kind int,
	pubkeys []string,
) []*dataloader.Result[[]*nostr.Event] {
	ctx, cancel := context.WithTimeoutCause(context.Background(), time.Second*6,
		errors.New("batch addressable load took too long"),
	)
	defer cancel()

	batchSize := len(pubkeys)
	results := make([]*dataloader.Result[[]*nostr.Event], batchSize)
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
				results[i] = &dataloader.Result[[]*nostr.Event]{
					Error: fmt.Errorf("won't proceed to query relays with a shortened key (%d)", kind),
				}
				return
			}

			// save attempts here so we don't try the same failed query over and over
			if doItNow := doThisNotMoreThanOnceAnHour("repl:" + strconv.Itoa(kind) + pubkey); !doItNow {
				results[i] = &dataloader.Result[[]*nostr.Event]{
					Error: fmt.Errorf("last attempt failed, waiting more to try again"),
				}
				return
			}

			// gather relays we'll use for this pubkey
			relays := sys.determineRelaysToQuery(pubkey, kind)

			// by default we will return an error (this will be overwritten when we find an event)
			results[i] = &dataloader.Result[[]*nostr.Event]{
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

	// wait for relay batches to be prepared
	wg.Wait()

	// query all relays with the prepared filters
	multiSubs := sys.Pool.BatchedSubManyEose(ctx, relayFilter)
nextEvent:
	for {
		select {
		case ie, more := <-multiSubs:
			if !more {
				return results
			}

			// insert this event at the desired position
			pos := keyPositions[ie.PubKey] // @unchecked: it must succeed because it must be a key we passed

			events := results[pos].Data
			if events == nil {
				// no events found, so just add this and end
				results[pos] = &dataloader.Result[[]*nostr.Event]{Data: []*nostr.Event{ie.Event}}
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

			// there is no match, so add to the end
			events = append(events, ie.Event)
			results[pos].Data = events
		case <-ctx.Done():
			return results
		}
	}
}
