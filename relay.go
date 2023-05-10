package nostr

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	s "github.com/SaveTheRbtz/generic-sync-map-go"
)

type Status int

const (
	PublishStatusSent      Status = 0
	PublishStatusFailed    Status = -1
	PublishStatusSucceeded Status = 1
)

var subscriptionIdCounter = 0

func (s Status) String() string {
	switch s {
	case PublishStatusSent:
		return "sent"
	case PublishStatusFailed:
		return "failed"
	case PublishStatusSucceeded:
		return "success"
	}

	return "unknown"
}

type Relay struct {
	URL           string
	RequestHeader http.Header // e.g. for origin header

	Connection    *Connection
	subscriptions s.MapOf[string, *Subscription]

	Challenges              chan string // NIP-42 Challenges
	Notices                 chan string
	ConnectionError         error
	connectionContext       context.Context // will be canceled when the connection closes
	connectionContextCancel context.CancelFunc

	okCallbacks s.MapOf[string, func(bool, string)]
	mutex       sync.RWMutex

	// custom things that aren't often used
	//
	AssumeValid bool // this will skip verifying signatures for events received from this relay
}

// NewRelay returns a new relay. The relay connection will be closed when the context is canceled.
func NewRelay(ctx context.Context, url string) *Relay {
	return &Relay{URL: NormalizeURL(url), connectionContext: ctx}
}

// RelayConnect returns a relay object connected to url.
// Once successfully connected, cancelling ctx has no effect.
// To close the connection, call r.Close().
func RelayConnect(ctx context.Context, url string) (*Relay, error) {
	r := NewRelay(context.Background(), url)
	err := r.Connect(ctx)
	return r, err
}

func (r *Relay) String() string {
	return r.URL
}

// Context retrieves the context that is associated with this relay connection.
func (r *Relay) Context() context.Context { return r.connectionContext }

// Connect tries to establish a websocket connection to r.URL.
// If the context expires before the connection is complete, an error is returned.
// Once successfully connected, context expiration has no effect: call r.Close
// to close the connection.
//
// The underlying relay connection will use a background context. If you want to
// pass a custom context to the underlying relay connection, use NewRelay() and
// then Relay.Connect().
func (r *Relay) Connect(ctx context.Context) error {
	if r.connectionContext == nil {
		connectionContext, cancel := context.WithCancel(context.Background())
		r.connectionContext = connectionContext
		r.connectionContextCancel = cancel
	}

	if r.URL == "" {
		return fmt.Errorf("invalid relay URL '%s'", r.URL)
	}

	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 7*time.Second)
		defer cancel()
	}

	conn, err := NewConnection(ctx, r.URL, r.RequestHeader)
	if err != nil {
		return fmt.Errorf("error opening websocket to '%s': %w", r.URL, err)
	}
	r.Connection = conn

	r.Challenges = make(chan string)
	r.Notices = make(chan string)

	// close these channels when the connection is dropped
	go func() {
		<-r.connectionContext.Done()
		r.mutex.Lock()
		close(r.Challenges)
		close(r.Notices)
		r.mutex.Unlock()
	}()

	// ping every 29 seconds
	go func() {
		ticker := time.NewTicker(29 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := conn.Ping()
				if err != nil {
					InfoLogger.Printf("{%s} error writing ping: %v; closing websocket", r.URL, err)
					return
				}
			}
		}
	}()

	// handling received messages
	go func() {
		for {
			message, err := conn.ReadMessage(r.connectionContext)
			if err != nil {
				r.ConnectionError = err
				break
			}

			envelope := ParseMessage(message)
			if envelope == nil {
				continue
			}

			switch env := envelope.(type) {
			case *NoticeEnvelope:
				debugLog("{%s} %v\n", r.URL, message)
				// TODO: improve this, otherwise if the application doesn't read the notices
				//       we'll consume ever more memory with each new notice
				go func() {
					r.mutex.RLock()
					if r.connectionContext.Err() == nil {
						r.Notices <- string(*env)
					}
					r.mutex.RUnlock()
				}()
			case *AuthEnvelope:
				debugLog("{%s} %v\n", r.URL, message)
				if env.Challenge == nil {
					continue
				}
				// TODO: same as with NoticeEnvelope
				go func() {
					r.mutex.RLock()
					if r.connectionContext.Err() == nil {
						r.Challenges <- *env.Challenge
					}
					r.mutex.RUnlock()
				}()
			case *EventEnvelope:
				debugLog("{%s} %v\n", r.URL, message)
				if env.SubscriptionID == nil {
					continue
				}
				if subscription, ok := r.subscriptions.Load(*env.SubscriptionID); !ok {
					InfoLogger.Printf("{%s} no subscription with id '%s'\n", r.URL, *env.SubscriptionID)
					continue
				} else {
					func() {
						// check if the event matches the desired filter, ignore otherwise
						if !subscription.Filters.Match(&env.Event) {
							InfoLogger.Printf("{%s} filter does not match: %v ~ %v\n", r.URL, subscription.Filters[0], env.Event)
							return
						}

						subscription.mutex.Lock()
						defer subscription.mutex.Unlock()
						if subscription.stopped {
							return
						}

						// check signature, ignore invalid, except from trusted (AssumeValid) relays
						if !r.AssumeValid {
							if ok, err := env.Event.CheckSignature(); !ok {
								errmsg := ""
								if err != nil {
									errmsg = err.Error()
								}
								InfoLogger.Printf("{%s} bad signature: %s\n", r.URL, errmsg)
								return
							}
						}

						subscription.Events <- &env.Event
					}()
				}
			case *EOSEEnvelope:
				debugLog("{%s} %v\n", r.URL, message)
				if subscription, ok := r.subscriptions.Load(string(*env)); ok {
					subscription.emitEose.Do(func() {
						subscription.EndOfStoredEvents <- struct{}{}
					})
				}
			case *OKEnvelope:
				if okCallback, exist := r.okCallbacks.Load(env.EventID); exist {
					okCallback(env.OK, *env.Reason)
				}
			}
		}
	}()

	return nil
}

// Publish sends an "EVENT" command to the relay r as in NIP-01.
// Status can be: success, failed, or sent (no response from relay before ctx times out).
func (r *Relay) Publish(ctx context.Context, event Event) (Status, error) {
	status := PublishStatusFailed
	var err error

	// data races on status variable without this mutex
	var mu sync.Mutex

	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 7*time.Second)
		defer cancel()
	}

	// make it cancellable so we can stop everything upon receiving an "OK"
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	// listen for an OK callback
	okCallback := func(ok bool, msg string) {
		mu.Lock()
		defer mu.Unlock()
		if ok {
			status = PublishStatusSucceeded
		} else {
			status = PublishStatusFailed
			err = fmt.Errorf("msg: %s", msg)
		}
		cancel()
	}
	r.okCallbacks.Store(event.ID, okCallback)
	defer r.okCallbacks.Delete(event.ID)

	// publish event
	envb, _ := EventEnvelope{Event: event}.MarshalJSON()
	debugLog("{%s} sending %v\n", r.URL, envb)
	status = PublishStatusSent
	if err := r.Connection.WriteMessage(envb); err != nil {
		status = PublishStatusFailed
		return status, err
	}

	sub := r.PrepareSubscription(ctx)
	sub.SetLabel("publish-check")
	sub.Filters = Filters{Filter{IDs: []string{event.ID}}}

	for {
		select {
		case <-ctx.Done(): // this will be called when we get an OK
			// proceed to return status as it is
			// e.g. if this happens because of the timeout then status will probably be "failed"
			//      but if it happens because okCallback was called then it might be "succeeded"
			// do not return if okCallback is in process
			return status, err
		case <-r.connectionContext.Done():
			// same as above, but when the relay loses connectivity entirely
			return status, err
		case <-time.After(4 * time.Second):
			// if we don't get an OK after 4 seconds, try to subscribe to the event
			if err := sub.Fire(); err != nil {
				InfoLogger.Printf("failed to subscribe to just published event %s at %s: %s", event.ID, r.URL, err)
			}
		case receivedEvent := <-sub.Events:
			if receivedEvent == nil {
				// channel is closed
				return status, err
			}

			if receivedEvent.ID == event.ID {
				// we got a success, so update our status and proceed to return
				mu.Lock()
				status = PublishStatusSucceeded
				mu.Unlock()
				return status, err
			}
		}
	}
}

// Auth sends an "AUTH" command client -> relay as in NIP-42.
// Status can be: success, failed, or sent (no response from relay before ctx times out).
func (r *Relay) Auth(ctx context.Context, event Event) (Status, error) {
	status := PublishStatusFailed
	var err error

	// data races on status variable without this mutex
	var mu sync.Mutex

	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 3 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}

	// make it cancellable so we can stop everything upon receiving an "OK"
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	// listen for an OK callback
	okCallback := func(ok bool, msg string) {
		mu.Lock()
		if ok {
			status = PublishStatusSucceeded
		} else {
			status = PublishStatusFailed
			err = fmt.Errorf("msg: %s", msg)
		}
		mu.Unlock()
		cancel()
	}
	r.okCallbacks.Store(event.ID, okCallback)
	defer r.okCallbacks.Delete(event.ID)

	// send AUTH
	authResponse, _ := AuthEnvelope{Event: event}.MarshalJSON()
	debugLog("{%s} sending %v\n", r.URL, authResponse)
	if err := r.Connection.WriteMessage(authResponse); err != nil {
		// status will be "failed"
		return status, err
	}
	// use mu.Lock() just in case the okCallback got called, extremely unlikely.
	mu.Lock()
	status = PublishStatusSent
	mu.Unlock()

	// the context either times out, and the status is "sent"
	// or the okCallback is called and the status is set to "succeeded" or "failed"
	// NIP-42 does not mandate an "OK" reply to an "AUTH" message
	<-ctx.Done()
	mu.Lock()
	defer mu.Unlock()
	return status, err
}

// Subscribe sends a "REQ" command to the relay r as in NIP-01.
// Events are returned through the channel sub.Events.
// The subscription is closed when context ctx is cancelled ("CLOSE" in NIP-01).
func (r *Relay) Subscribe(ctx context.Context, filters Filters) (*Subscription, error) {
	if r.Connection == nil {
		panic(fmt.Errorf("must call .Connect() first before calling .Subscribe()"))
	}

	sub := r.PrepareSubscription(ctx)
	sub.Filters = filters

	if err := sub.Fire(); err != nil {
		return nil, fmt.Errorf("couldn't subscribe to %v at %s: %w", filters, r.URL, err)
	}

	return sub, nil
}

func (r *Relay) QuerySync(ctx context.Context, filter Filter) ([]*Event, error) {
	sub, err := r.Subscribe(ctx, Filters{filter})
	if err != nil {
		return nil, err
	}

	defer sub.Unsub()

	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 3 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 7*time.Second)
		defer cancel()
	}

	var events []*Event
	for {
		select {
		case evt := <-sub.Events:
			if evt == nil {
				// channel is closed
				return events, nil
			}
			events = append(events, evt)
		case <-sub.EndOfStoredEvents:
			return events, nil
		case <-ctx.Done():
			return events, nil
		}
	}
}

func (r *Relay) PrepareSubscription(ctx context.Context) *Subscription {
	current := subscriptionIdCounter
	subscriptionIdCounter++

	ctx, cancel := context.WithCancel(ctx)

	return &Subscription{
		Relay:             r,
		Context:           ctx,
		cancel:            cancel,
		conn:              r.Connection,
		counter:           current,
		Events:            make(chan *Event),
		EndOfStoredEvents: make(chan struct{}, 1),
	}
}

func (r *Relay) Close() error {
	if r.connectionContextCancel == nil {
		return fmt.Errorf("relay not connected")
	}

	r.connectionContextCancel()
	r.connectionContextCancel = nil
	return r.Connection.Close()
}
