package sdk

import (
	"context"
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
		dataloader.WithBatchCapacity[string, []*nostr.Event](60),
		dataloader.WithClearCacheOnBatch[string, []*nostr.Event](),
		dataloader.WithWait[string, []*nostr.Event](time.Millisecond*350),
	)
}

func (sys *System) batchLoadAddressableEvents(
	kind int,
	pubkeys []string,
) []*dataloader.Result[[]*nostr.Event] {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
	defer cancel()

	batchSize := len(pubkeys)
	results := make([]*dataloader.Result[[]*nostr.Event], batchSize)
	keyPositions := make(map[string]int)          // { [pubkey]: slice_index }
	relayFilters := make(map[string]nostr.Filter) // { [relayUrl]: filter }

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
			relays := sys.determineRelaysToQuery(ctx, pubkey, kind)

			// by default we will return an error (this will be overwritten when we find an event)
			results[i] = &dataloader.Result[[]*nostr.Event]{
				Error: fmt.Errorf("couldn't find a kind %d event anywhere %v", kind, relays),
			}

			cm.Lock()
			for _, relay := range relays {
				// each relay will have a custom filter
				filter, ok := relayFilters[relay]
				if !ok {
					filter = nostr.Filter{
						Kinds:   []int{kind},
						Authors: make([]string, 0, batchSize-i /* this and all pubkeys after this can be added */),
					}
				}
				filter.Authors = append(filter.Authors, pubkey)
				relayFilters[relay] = filter
			}
			cm.Unlock()
		}(i, pubkey)
	}

	// query all relays with the prepared filters
	wg.Wait()
	multiSubs := sys.batchAddressableRelayQueries(ctx, relayFilters)
nextEvent:
	for {
		select {
		case evt, more := <-multiSubs:
			if !more {
				return results
			}

			// insert this event at the desired position
			pos := keyPositions[evt.PubKey] // @unchecked: it must succeed because it must be a key we passed

			events := results[pos].Data
			if events == nil {
				// no events found, so just add this and end
				results[pos] = &dataloader.Result[[]*nostr.Event]{Data: []*nostr.Event{evt}}
				continue nextEvent
			}

			// there are events, so look for a match
			d := evt.Tags.GetD()
			for i, event := range events {
				if event.Tags.GetD() == d {
					// there is a match
					if event.CreatedAt < evt.CreatedAt {
						// ...and this one is newer, so replace
						events[i] = evt
					} else {
						// ... but this one is older, so ignore
					}
					// in any case we end this here
					continue nextEvent
				}
			}

			// there is no match, so add to the end
			events = append(events, evt)
			results[pos].Data = events
		case <-ctx.Done():
			return results
		}
	}
}

// batchAddressableRelayQueries is like batchReplaceableRelayQueries, except it doesn't count results to
// try to exit early.
func (sys *System) batchAddressableRelayQueries(
	ctx context.Context,
	relayFilters map[string]nostr.Filter,
) <-chan *nostr.Event {
	all := make(chan *nostr.Event)

	wg := sync.WaitGroup{}
	wg.Add(len(relayFilters))
	for url, filter := range relayFilters {
		go func(url string, filter nostr.Filter) {
			defer wg.Done()
			n := len(filter.Authors)

			ctx, cancel := context.WithTimeout(ctx, time.Millisecond*450+time.Millisecond*50*time.Duration(n))
			defer cancel()

			for ie := range sys.Pool.SubManyEose(ctx, []string{url}, nostr.Filters{filter}, nostr.WithLabel("addr")) {
				all <- ie.Event
			}
		}(url, filter)
	}

	go func() {
		wg.Wait()
		close(all)
	}()

	return all
}
