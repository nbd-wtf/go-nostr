package nostr

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nbd-wtf/go-nostr/nip45/hyperloglog"
	"github.com/puzpuzpuz/xsync/v3"
)

const (
	seenAlreadyDropTick = time.Minute
)

// SimplePool manages connections to multiple relays, ensures they are reopened when necessary and not duplicated.
type SimplePool struct {
	Relays  *xsync.MapOf[string, *Relay]
	Context context.Context

	authHandler func(context.Context, RelayEvent) error
	cancel      context.CancelCauseFunc

	eventMiddleware     func(RelayEvent)
	duplicateMiddleware func(relay string, id string)
	queryMiddleware     func(relay string, pubkey string, kind int)

	// custom things not often used
	penaltyBoxMu sync.Mutex
	penaltyBox   map[string][2]float64
	relayOptions []RelayOption
}

// DirectedFilter combines a Filter with a specific relay URL.
type DirectedFilter struct {
	Filter
	Relay string
}

// RelayEvent represents an event received from a specific relay.
type RelayEvent struct {
	*Event
	Relay *Relay
}

func (ie RelayEvent) String() string { return fmt.Sprintf("[%s] >> %s", ie.Relay.URL, ie.Event) }

// PoolOption is an interface for options that can be applied to a SimplePool.
type PoolOption interface {
	ApplyPoolOption(*SimplePool)
}

// NewSimplePool creates a new SimplePool with the given context and options.
func NewSimplePool(ctx context.Context, opts ...PoolOption) *SimplePool {
	ctx, cancel := context.WithCancelCause(ctx)

	pool := &SimplePool{
		Relays: xsync.NewMapOf[string, *Relay](),

		Context: ctx,
		cancel:  cancel,
	}

	for _, opt := range opts {
		opt.ApplyPoolOption(pool)
	}

	return pool
}

// WithRelayOptions sets options that will be used on every relay instance created by this pool.
func WithRelayOptions(ropts ...RelayOption) withRelayOptionsOpt {
	return ropts
}

type withRelayOptionsOpt []RelayOption

func (h withRelayOptionsOpt) ApplyPoolOption(pool *SimplePool) {
	pool.relayOptions = h
}

// WithAuthHandler must be a function that signs the auth event when called.
// it will be called whenever any relay in the pool returns a `CLOSED` message
// with the "auth-required:" prefix, only once for each relay
type WithAuthHandler func(ctx context.Context, authEvent RelayEvent) error

func (h WithAuthHandler) ApplyPoolOption(pool *SimplePool) {
	pool.authHandler = h
}

// WithPenaltyBox just sets the penalty box mechanism so relays that fail to connect
// or that disconnect will be ignored for a while and we won't attempt to connect again.
func WithPenaltyBox() withPenaltyBoxOpt { return withPenaltyBoxOpt{} }

type withPenaltyBoxOpt struct{}

func (h withPenaltyBoxOpt) ApplyPoolOption(pool *SimplePool) {
	pool.penaltyBox = make(map[string][2]float64)
	go func() {
		sleep := 30.0
		for {
			time.Sleep(time.Duration(sleep) * time.Second)

			pool.penaltyBoxMu.Lock()
			nextSleep := 300.0
			for url, v := range pool.penaltyBox {
				remainingSeconds := v[1]
				remainingSeconds -= sleep
				if remainingSeconds <= 0 {
					pool.penaltyBox[url] = [2]float64{v[0], 0}
					continue
				} else {
					pool.penaltyBox[url] = [2]float64{v[0], remainingSeconds}
				}

				if remainingSeconds < nextSleep {
					nextSleep = remainingSeconds
				}
			}

			sleep = nextSleep
			pool.penaltyBoxMu.Unlock()
		}
	}()
}

// WithEventMiddleware is a function that will be called with all events received.
type WithEventMiddleware func(RelayEvent)

func (h WithEventMiddleware) ApplyPoolOption(pool *SimplePool) {
	pool.eventMiddleware = h
}

// WithDuplicateMiddleware is a function that will be called with all duplicate ids received.
type WithDuplicateMiddleware func(relay string, id string)

func (h WithDuplicateMiddleware) ApplyPoolOption(pool *SimplePool) {
	pool.duplicateMiddleware = h
}

// WithAuthorKindQueryMiddleware is a function that will be called with every combination of relay+pubkey+kind queried
// in a .SubMany*() call -- when applicable (i.e. when the query contains a pubkey and a kind).
type WithAuthorKindQueryMiddleware func(relay string, pubkey string, kind int)

func (h WithAuthorKindQueryMiddleware) ApplyPoolOption(pool *SimplePool) {
	pool.queryMiddleware = h
}

var (
	_ PoolOption = (WithAuthHandler)(nil)
	_ PoolOption = (WithEventMiddleware)(nil)
	_ PoolOption = WithPenaltyBox()
	_ PoolOption = WithRelayOptions(WithRequestHeader(http.Header{}))
)

// EnsureRelay ensures that a relay connection exists and is active.
// If the relay is not connected, it attempts to connect.
func (pool *SimplePool) EnsureRelay(url string) (*Relay, error) {
	nm := NormalizeURL(url)
	defer namedLock(nm)()

	relay, ok := pool.Relays.Load(nm)
	if ok && relay == nil {
		if pool.penaltyBox != nil {
			pool.penaltyBoxMu.Lock()
			defer pool.penaltyBoxMu.Unlock()
			v, _ := pool.penaltyBox[nm]
			if v[1] > 0 {
				return nil, fmt.Errorf("in penalty box, %fs remaining", v[1])
			}
		}
	} else if ok && relay.IsConnected() {
		// already connected, unlock and return
		return relay, nil
	}

	// try to connect
	// we use this ctx here so when the pool dies everything dies
	ctx, cancel := context.WithTimeoutCause(
		pool.Context,
		time.Second*15,
		errors.New("connecting to the relay took too long"),
	)
	defer cancel()

	relay = NewRelay(context.Background(), url, pool.relayOptions...)
	if err := relay.Connect(ctx); err != nil {
		if pool.penaltyBox != nil {
			// putting relay in penalty box
			pool.penaltyBoxMu.Lock()
			defer pool.penaltyBoxMu.Unlock()
			v, _ := pool.penaltyBox[nm]
			pool.penaltyBox[nm] = [2]float64{v[0] + 1, 30.0 + math.Pow(2, v[0]+1)}
		}
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	pool.Relays.Store(nm, relay)
	return relay, nil
}

// PublishResult represents the result of publishing an event to a relay.
type PublishResult struct {
	Error    error
	RelayURL string
	Relay    *Relay
}

// PublishMany publishes an event to multiple relays and returns a channel of results emitted as they're received.
func (pool *SimplePool) PublishMany(ctx context.Context, urls []string, evt Event) chan PublishResult {
	ch := make(chan PublishResult, len(urls))

	wg := sync.WaitGroup{}
	wg.Add(len(urls))
	go func() {
		for _, url := range urls {
			go func() {
				defer wg.Done()

				relay, err := pool.EnsureRelay(url)
				if err != nil {
					ch <- PublishResult{err, url, nil}
					return
				}

				if err := relay.Publish(ctx, evt); err == nil {
					// success with no auth required
					ch <- PublishResult{nil, url, relay}
				} else if strings.HasPrefix(err.Error(), "msg: auth-required:") && pool.authHandler != nil {
					// try to authenticate if we can
					if authErr := relay.Auth(ctx, func(event *Event) error {
						return pool.authHandler(ctx, RelayEvent{Event: event, Relay: relay})
					}); authErr == nil {
						if err := relay.Publish(ctx, evt); err == nil {
							// success after auth
							ch <- PublishResult{nil, url, relay}
						} else {
							// failure after auth
							ch <- PublishResult{err, url, relay}
						}
					} else {
						// failure to auth
						ch <- PublishResult{fmt.Errorf("failed to auth: %w", authErr), url, relay}
					}
				} else {
					// direct failure
					ch <- PublishResult{err, url, relay}
				}
			}()
		}

		wg.Wait()
		close(ch)
	}()

	return ch
}

// SubscribeMany opens a subscription with the given filter to multiple relays
// the subscriptions ends when the context is canceled or when all relays return a CLOSED.
func (pool *SimplePool) SubscribeMany(
	ctx context.Context,
	urls []string,
	filter Filter,
	opts ...SubscriptionOption,
) chan RelayEvent {
	return pool.subMany(ctx, urls, Filters{filter}, nil, opts...)
}

// FetchMany opens a subscription, much like SubscribeMany, but it ends as soon as all Relays
// return an EOSE message.
func (pool *SimplePool) FetchMany(
	ctx context.Context,
	urls []string,
	filter Filter,
	opts ...SubscriptionOption,
) chan RelayEvent {
	return pool.SubManyEose(ctx, urls, Filters{filter}, opts...)
}

// Deprecated: SubMany is deprecated: use SubscribeMany instead.
func (pool *SimplePool) SubMany(
	ctx context.Context,
	urls []string,
	filters Filters,
	opts ...SubscriptionOption,
) chan RelayEvent {
	return pool.subMany(ctx, urls, filters, nil, opts...)
}

// SubscribeManyNotifyEOSE is like SubscribeMany, but takes a channel that is closed when
// all subscriptions have received an EOSE
func (pool *SimplePool) SubscribeManyNotifyEOSE(
	ctx context.Context,
	urls []string,
	filter Filter,
	eoseChan chan struct{},
	opts ...SubscriptionOption,
) chan RelayEvent {
	return pool.subMany(ctx, urls, Filters{filter}, eoseChan, opts...)
}

type ReplaceableKey struct {
	PubKey string
	D      string
}

// FetchManyReplaceable is like FetchMany, but deduplicates replaceable and addressable events and returns
// only the latest for each "d" tag.
func (pool *SimplePool) FetchManyReplaceable(
	ctx context.Context,
	urls []string,
	filter Filter,
	opts ...SubscriptionOption,
) *xsync.MapOf[ReplaceableKey, *Event] {
	ctx, cancel := context.WithCancelCause(ctx)

	results := xsync.NewMapOf[ReplaceableKey, *Event]()

	wg := sync.WaitGroup{}
	wg.Add(len(urls))

	seenAlreadyLatest := xsync.NewMapOf[ReplaceableKey, Timestamp]()
	opts = append(opts, WithCheckDuplicateReplaceable(func(rk ReplaceableKey, ts Timestamp) bool {
		updated := false
		seenAlreadyLatest.Compute(rk, func(latest Timestamp, _ bool) (newValue Timestamp, delete bool) {
			if ts > latest {
				updated = true // we are updating the most recent
				return ts, false
			}
			return latest, false // the one we had was already more recent
		})
		return updated
	}))

	for _, url := range urls {
		go func(nm string) {
			defer wg.Done()

			if mh := pool.queryMiddleware; mh != nil {
				if filter.Kinds != nil && filter.Authors != nil {
					for _, kind := range filter.Kinds {
						for _, author := range filter.Authors {
							mh(nm, author, kind)
						}
					}
				}
			}

			relay, err := pool.EnsureRelay(nm)
			if err != nil {
				debugLogf("error connecting to %s with %v: %s", nm, filter, err)
				return
			}

			hasAuthed := false

		subscribe:
			sub, err := relay.Subscribe(ctx, Filters{filter}, opts...)
			if err != nil {
				debugLogf("error subscribing to %s with %v: %s", relay, filter, err)
				return
			}

			for {
				select {
				case <-ctx.Done():
					return
				case <-sub.EndOfStoredEvents:
					return
				case reason := <-sub.ClosedReason:
					if strings.HasPrefix(reason, "auth-required:") && pool.authHandler != nil && !hasAuthed {
						// relay is requesting auth. if we can we will perform auth and try again
						err := relay.Auth(ctx, func(event *Event) error {
							return pool.authHandler(ctx, RelayEvent{Event: event, Relay: relay})
						})
						if err == nil {
							hasAuthed = true // so we don't keep doing AUTH again and again
							goto subscribe
						}
					}
					debugLogf("CLOSED from %s: '%s'\n", nm, reason)
					return
				case evt, more := <-sub.Events:
					if !more {
						return
					}

					ie := RelayEvent{Event: evt, Relay: relay}
					if mh := pool.eventMiddleware; mh != nil {
						mh(ie)
					}

					results.Store(ReplaceableKey{evt.PubKey, evt.Tags.GetD()}, evt)
				}
			}
		}(NormalizeURL(url))
	}

	// this will happen when all subscriptions get an eose (or when they die)
	wg.Wait()
	cancel(errors.New("all subscriptions ended"))

	return results
}

func (pool *SimplePool) subMany(
	ctx context.Context,
	urls []string,
	filters Filters,
	eoseChan chan struct{},
	opts ...SubscriptionOption,
) chan RelayEvent {
	ctx, cancel := context.WithCancelCause(ctx)
	_ = cancel // do this so `go vet` will stop complaining
	events := make(chan RelayEvent)
	seenAlready := xsync.NewMapOf[string, Timestamp]()
	ticker := time.NewTicker(seenAlreadyDropTick)

	eoseWg := sync.WaitGroup{}
	eoseWg.Add(len(urls))
	if eoseChan != nil {
		go func() {
			eoseWg.Wait()
			close(eoseChan)
		}()
	}

	pending := xsync.NewCounter()
	pending.Add(int64(len(urls)))
	for i, url := range urls {
		url = NormalizeURL(url)
		urls[i] = url
		if idx := slices.Index(urls, url); idx != i {
			// skip duplicate relays in the list
			eoseWg.Done()
			continue
		}

		eosed := atomic.Bool{}
		firstConnection := true

		go func(nm string) {
			defer func() {
				pending.Dec()
				if pending.Value() == 0 {
					close(events)
					cancel(fmt.Errorf("aborted: %w", context.Cause(ctx)))
				}
				if eosed.CompareAndSwap(false, true) {
					eoseWg.Done()
				}
			}()

			hasAuthed := false
			interval := 3 * time.Second
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				var sub *Subscription

				if mh := pool.queryMiddleware; mh != nil {
					for _, filter := range filters {
						if filter.Kinds != nil && filter.Authors != nil {
							for _, kind := range filter.Kinds {
								for _, author := range filter.Authors {
									mh(nm, author, kind)
								}
							}
						}
					}
				}

				relay, err := pool.EnsureRelay(nm)
				if err != nil {
					// if we never connected to this just fail
					if firstConnection {
						return
					}

					// otherwise (if we were connected and got disconnected) keep trying to reconnect
					debugLogf("%s reconnecting because connection failed\n", nm)
					goto reconnect
				}
				firstConnection = false
				hasAuthed = false

			subscribe:
				sub, err = relay.Subscribe(ctx, filters, append(opts, WithCheckDuplicate(func(id, relay string) bool {
					_, exists := seenAlready.Load(id)
					if exists && pool.duplicateMiddleware != nil {
						pool.duplicateMiddleware(relay, id)
					}
					return exists
				}))...)
				if err != nil {
					debugLogf("%s reconnecting because subscription died\n", nm)
					goto reconnect
				}

				go func() {
					<-sub.EndOfStoredEvents

					// guard here otherwise a resubscription will trigger a duplicate call to eoseWg.Done()
					if eosed.CompareAndSwap(false, true) {
						eoseWg.Done()
					}
				}()

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
							debugLogf("%s reconnecting because sub.Events is broken\n", nm)
							goto reconnect
						}

						ie := RelayEvent{Event: evt, Relay: relay}
						if mh := pool.eventMiddleware; mh != nil {
							mh(ie)
						}

						select {
						case events <- ie:
						case <-ctx.Done():
							return
						}
					case <-ticker.C:
						if eosed.Load() {
							old := Timestamp(time.Now().Add(-seenAlreadyDropTick).Unix())
							for id, value := range seenAlready.Range {
								if value < old {
									seenAlready.Delete(id)
								}
							}
						}
					case reason := <-sub.ClosedReason:
						if strings.HasPrefix(reason, "auth-required:") && pool.authHandler != nil && !hasAuthed {
							// relay is requesting auth. if we can we will perform auth and try again
							err := relay.Auth(ctx, func(event *Event) error {
								return pool.authHandler(ctx, RelayEvent{Event: event, Relay: relay})
							})
							if err == nil {
								hasAuthed = true // so we don't keep doing AUTH again and again
								goto subscribe
							}
						} else {
							debugLogf("CLOSED from %s: '%s'\n", nm, reason)
						}

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
		}(url)
	}

	return events
}

// Deprecated: SubManyEose is deprecated: use FetchMany instead.
func (pool *SimplePool) SubManyEose(
	ctx context.Context,
	urls []string,
	filters Filters,
	opts ...SubscriptionOption,
) chan RelayEvent {
	seenAlready := xsync.NewMapOf[string, struct{}]()
	return pool.subManyEoseNonOverwriteCheckDuplicate(ctx, urls, filters,
		WithCheckDuplicate(func(id, relay string) bool {
			_, exists := seenAlready.LoadOrStore(id, struct{}{})
			if exists && pool.duplicateMiddleware != nil {
				pool.duplicateMiddleware(relay, id)
			}
			return exists
		}),
		opts...)
}

func (pool *SimplePool) subManyEoseNonOverwriteCheckDuplicate(
	ctx context.Context,
	urls []string,
	filters Filters,
	wcd WithCheckDuplicate,
	opts ...SubscriptionOption,
) chan RelayEvent {
	ctx, cancel := context.WithCancelCause(ctx)

	events := make(chan RelayEvent)
	wg := sync.WaitGroup{}
	wg.Add(len(urls))

	opts = append(opts, wcd)

	go func() {
		// this will happen when all subscriptions get an eose (or when they die)
		wg.Wait()
		cancel(errors.New("all subscriptions ended"))
		close(events)
	}()

	for _, url := range urls {
		go func(nm string) {
			defer wg.Done()

			if mh := pool.queryMiddleware; mh != nil {
				for _, filter := range filters {
					if filter.Kinds != nil && filter.Authors != nil {
						for _, kind := range filter.Kinds {
							for _, author := range filter.Authors {
								mh(nm, author, kind)
							}
						}
					}
				}
			}

			relay, err := pool.EnsureRelay(nm)
			if err != nil {
				debugLogf("error connecting to %s with %v: %s", nm, filters, err)
				return
			}

			hasAuthed := false

		subscribe:
			sub, err := relay.Subscribe(ctx, filters, opts...)
			if err != nil {
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
					if strings.HasPrefix(reason, "auth-required:") && pool.authHandler != nil && !hasAuthed {
						// relay is requesting auth. if we can we will perform auth and try again
						err := relay.Auth(ctx, func(event *Event) error {
							return pool.authHandler(ctx, RelayEvent{Event: event, Relay: relay})
						})
						if err == nil {
							hasAuthed = true // so we don't keep doing AUTH again and again
							goto subscribe
						}
					}
					debugLogf("CLOSED from %s: '%s'\n", nm, reason)
					return
				case evt, more := <-sub.Events:
					if !more {
						return
					}

					ie := RelayEvent{Event: evt, Relay: relay}
					if mh := pool.eventMiddleware; mh != nil {
						mh(ie)
					}

					select {
					case events <- ie:
					case <-ctx.Done():
						return
					}
				}
			}
		}(NormalizeURL(url))
	}

	return events
}

// CountMany aggregates count results from multiple relays using NIP-45 HyperLogLog
func (pool *SimplePool) CountMany(
	ctx context.Context,
	urls []string,
	filter Filter,
	opts []SubscriptionOption,
) int {
	hll := hyperloglog.New(0) // offset is irrelevant here

	wg := sync.WaitGroup{}
	wg.Add(len(urls))
	for _, url := range urls {
		go func(nm string) {
			defer wg.Done()
			relay, err := pool.EnsureRelay(url)
			if err != nil {
				return
			}
			ce, err := relay.countInternal(ctx, Filters{filter}, opts...)
			if err != nil {
				return
			}
			if len(ce.HyperLogLog) != 256 {
				return
			}
			hll.MergeRegisters(ce.HyperLogLog)
		}(NormalizeURL(url))
	}

	wg.Wait()
	return int(hll.Count())
}

// QuerySingle returns the first event returned by the first relay, cancels everything else.
func (pool *SimplePool) QuerySingle(
	ctx context.Context,
	urls []string,
	filter Filter,
	opts ...SubscriptionOption,
) *RelayEvent {
	ctx, cancel := context.WithCancelCause(ctx)
	for ievt := range pool.SubManyEose(ctx, urls, Filters{filter}, opts...) {
		cancel(errors.New("got the first event and ended successfully"))
		return &ievt
	}
	cancel(errors.New("SubManyEose() didn't get yield events"))
	return nil
}

// BatchedSubManyEose performs batched subscriptions to multiple relays with different filters.
func (pool *SimplePool) BatchedSubManyEose(
	ctx context.Context,
	dfs []DirectedFilter,
	opts ...SubscriptionOption,
) chan RelayEvent {
	res := make(chan RelayEvent)
	wg := sync.WaitGroup{}
	wg.Add(len(dfs))
	seenAlready := xsync.NewMapOf[string, struct{}]()

	for _, df := range dfs {
		go func(df DirectedFilter) {
			for ie := range pool.subManyEoseNonOverwriteCheckDuplicate(ctx,
				[]string{df.Relay},
				Filters{df.Filter},
				WithCheckDuplicate(func(id, relay string) bool {
					_, exists := seenAlready.LoadOrStore(id, struct{}{})
					if exists && pool.duplicateMiddleware != nil {
						pool.duplicateMiddleware(relay, id)
					}
					return exists
				}), opts...,
			) {
				select {
				case res <- ie:
				case <-ctx.Done():
					wg.Done()
					return
				}
			}
			wg.Done()
		}(df)
	}

	go func() {
		wg.Wait()
		close(res)
	}()

	return res
}

// Close closes the pool with the given reason.
func (pool *SimplePool) Close(reason string) {
	pool.cancel(fmt.Errorf("pool closed with reason: '%s'", reason))
}
