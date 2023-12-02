package nostr

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/puzpuzpuz/xsync/v2"
)

const (
	seenAlreadyDropTick = time.Minute
)

type SimplePool struct {
	Relays  *xsync.MapOf[string, *Relay]
	Context context.Context

	cancel context.CancelFunc
}

type IncomingEvent struct {
	*Event
	Relay *Relay
}

func NewSimplePool(ctx context.Context) *SimplePool {
	ctx, cancel := context.WithCancel(ctx)

	return &SimplePool{
		Relays: xsync.NewMapOf[*Relay](),

		Context: ctx,
		cancel:  cancel,
	}
}

func (pool *SimplePool) EnsureRelay(url string) (*Relay, error) {
	nm := NormalizeURL(url)

	defer namedLock(url)()

	relay, ok := pool.Relays.Load(nm)
	if ok && relay.IsConnected() {
		// already connected, unlock and return
		return relay, nil
	} else {
		var err error
		// we use this ctx here so when the pool dies everything dies
		ctx, cancel := context.WithTimeout(pool.Context, time.Second*15)
		defer cancel()
		if relay, err = RelayConnect(ctx, nm); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}

		pool.Relays.Store(nm, relay)
		return relay, nil
	}
}

// SubMany opens a subscription with the given filters to multiple relays
// the subscriptions only end when the context is canceled
func (pool *SimplePool) SubMany(ctx context.Context, urls []string, filters Filters) chan IncomingEvent {
	return pool.subMany(ctx, urls, filters, true)
}

// SubManyNonUnique is like SubMany, but returns duplicate events if they come from different relays
func (pool *SimplePool) SubManyNonUnique(ctx context.Context, urls []string, filters Filters) chan IncomingEvent {
	return pool.subMany(ctx, urls, filters, false)
}

func (pool *SimplePool) subMany(ctx context.Context, urls []string, filters Filters, unique bool) chan IncomingEvent {
	ctx, cancel := context.WithCancel(ctx)
	_ = cancel // do this so `go vet` will stop complaining
	events := make(chan IncomingEvent)
	seenAlready := xsync.NewMapOf[Timestamp]()
	ticker := time.NewTicker(seenAlreadyDropTick)
	eose := false

	pending := xsync.NewCounter()
	pending.Add(int64(len(urls)))
	for _, url := range urls {
		go func(nm string) {
			defer func() {
				pending.Dec()
				if pending.Value() == 0 {
					close(events)
				}
				cancel()
			}()

			interval := 3 * time.Second
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				var sub *Subscription

				relay, err := pool.EnsureRelay(nm)
				if err != nil {
					goto reconnect
				}

				sub, err = relay.Subscribe(ctx, filters)
				if err != nil {
					goto reconnect
				}

				// reset interval when we get a good subscription
				interval = 3 * time.Second

				for {
					select {
					case evt, more := <-sub.Events:
						if !more {
							// this means the connection was closed for weird reasons, like the server shut down
							// so we will update the filters here to include only events seem from now on
							// and try to reconnect until we succeed
							now := Now()
							for i := range filters {
								filters[i].Since = &now
							}
							goto reconnect
						}
						if unique {
							if _, seen := seenAlready.LoadOrStore(evt.ID, evt.CreatedAt); seen {
								continue
							}
						}
						select {
						case events <- IncomingEvent{Event: evt, Relay: relay}:
						case <-ctx.Done():
						}
					case <-sub.EndOfStoredEvents:
						eose = true
					case <-ticker.C:
						if eose {
							old := Timestamp(time.Now().Add(-seenAlreadyDropTick).Unix())
							seenAlready.Range(func(id string, value Timestamp) bool {
								if value < old {
									seenAlready.Delete(id)
								}
								return true
							})
						}
					case reason := <-sub.ClosedReason:
						log.Printf("CLOSED from %s: '%s'\n", nm, reason)
						return
					case <-ctx.Done():
						return
					}
				}

			reconnect:
				// we will go back to the beginning of the loop and try to connect again and again
				// until the context is canceled
				time.Sleep(interval)
				interval = interval * 17 / 10 // the next time we try we will wait longer
			}
		}(NormalizeURL(url))
	}

	return events
}

// SubManyEose is like SubMany, but it stops subscriptions and closes the channel when gets a EOSE
func (pool *SimplePool) SubManyEose(ctx context.Context, urls []string, filters Filters) chan IncomingEvent {
	return pool.subManyEose(ctx, urls, filters, true)
}

// SubManyEoseNonUnique is like SubManyEose, but returns duplicate events if they come from different relays
func (pool *SimplePool) SubManyEoseNonUnique(ctx context.Context, urls []string, filters Filters) chan IncomingEvent {
	return pool.subManyEose(ctx, urls, filters, false)
}

func (pool *SimplePool) subManyEose(ctx context.Context, urls []string, filters Filters, unique bool) chan IncomingEvent {
	ctx, cancel := context.WithCancel(ctx)

	events := make(chan IncomingEvent)
	seenAlready := xsync.NewMapOf[bool]()
	wg := sync.WaitGroup{}
	wg.Add(len(urls))

	go func() {
		// this will happen when all subscriptions get an eose (or when they die)
		wg.Wait()
		cancel()
		close(events)
	}()

	for _, url := range urls {
		go func(nm string) {
			defer wg.Done()

			relay, err := pool.EnsureRelay(nm)
			if err != nil {
				return
			}

			sub, err := relay.Subscribe(ctx, filters)
			if sub == nil {
				debugLogf("error subscribing to %s with %v: %s", relay, filters, err)
				return
			}

			for {
				select {
				case <-ctx.Done():
					return
				case <-sub.EndOfStoredEvents:
					return
				case reason := <-sub.ClosedReason:
					log.Printf("CLOSED from %s: '%s'\n", nm, reason)
					return
				case evt, more := <-sub.Events:
					if !more {
						return
					}

					if unique {
						if _, seen := seenAlready.LoadOrStore(evt.ID, true); seen {
							continue
						}
					}

					select {
					case events <- IncomingEvent{Event: evt, Relay: relay}:
					case <-ctx.Done():
						return
					}
				}
			}
		}(NormalizeURL(url))
	}

	return events
}

// QuerySingle returns the first event returned by the first relay, cancels everything else.
func (pool *SimplePool) QuerySingle(ctx context.Context, urls []string, filter Filter) *IncomingEvent {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for ievt := range pool.SubManyEose(ctx, urls, Filters{filter}) {
		return &ievt
	}
	return nil
}
