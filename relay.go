package nostr

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/puzpuzpuz/xsync/v2"
)

type Status int

const (
	PublishStatusSent      Status = 0
	PublishStatusFailed    Status = -1
	PublishStatusSucceeded Status = 1
)

var subscriptionIDCounter atomic.Int32

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
	closeMutex sync.Mutex

	URL           string
	RequestHeader http.Header // e.g. for origin header

	Connection    *Connection
	Subscriptions *xsync.MapOf[string, *Subscription]

	ConnectionError         error
	connectionContext       context.Context // will be canceled when the connection closes
	connectionContextCancel context.CancelFunc

	challenge                     string      // NIP-42 challenge, we only keep the last
	notices                       chan string // NIP-01 NOTICEs
	okCallbacks                   *xsync.MapOf[string, func(bool, string)]
	writeQueue                    chan writeRequest
	subscriptionChannelCloseQueue chan *Subscription

	// custom things that aren't often used
	//
	AssumeValid bool // this will skip verifying signatures for events received from this relay
}

type writeRequest struct {
	msg    []byte
	answer chan error
}

// NewRelay returns a new relay. The relay connection will be closed when the context is canceled.
func NewRelay(ctx context.Context, url string, opts ...RelayOption) *Relay {
	ctx, cancel := context.WithCancel(ctx)
	r := &Relay{
		URL:                           NormalizeURL(url),
		connectionContext:             ctx,
		connectionContextCancel:       cancel,
		Subscriptions:                 xsync.NewMapOf[*Subscription](),
		okCallbacks:                   xsync.NewMapOf[func(bool, string)](),
		writeQueue:                    make(chan writeRequest),
		subscriptionChannelCloseQueue: make(chan *Subscription),
	}

	for _, opt := range opts {
		switch o := opt.(type) {
		case WithNoticeHandler:
			r.notices = make(chan string)
			go func() {
				for notice := range r.notices {
					o(notice)
				}
			}()
		}
	}

	return r
}

// RelayConnect returns a relay object connected to url.
// Once successfully connected, cancelling ctx has no effect.
// To close the connection, call r.Close().
func RelayConnect(ctx context.Context, url string, opts ...RelayOption) (*Relay, error) {
	r := NewRelay(context.Background(), url, opts...)
	err := r.Connect(ctx)
	return r, err
}

// When instantiating relay connections, some options may be passed.
// RelayOption is the type of the argument passed for that.
// Some examples of this are WithNoticeHandler and WithAuthHandler.
type RelayOption interface {
	IsRelayOption()
}

// WithNoticeHandler just takes notices and is expected to do something with them.
// when not given, defaults to logging the notices.
type WithNoticeHandler func(notice string)

func (_ WithNoticeHandler) IsRelayOption() {}

var _ RelayOption = (WithNoticeHandler)(nil)

// String just returns the relay URL.
func (r *Relay) String() string {
	return r.URL
}

// Context retrieves the context that is associated with this relay connection.
func (r *Relay) Context() context.Context { return r.connectionContext }

// IsConnected returns true if the connection to this relay seems to be active.
func (r *Relay) IsConnected() bool { return r.connectionContext.Err() == nil }

// Connect tries to establish a websocket connection to r.URL.
// If the context expires before the connection is complete, an error is returned.
// Once successfully connected, context expiration has no effect: call r.Close
// to close the connection.
//
// The underlying relay connection will use a background context. If you want to
// pass a custom context to the underlying relay connection, use NewRelay() and
// then Relay.Connect().
func (r *Relay) Connect(ctx context.Context) error {
	if r.connectionContext == nil || r.Subscriptions == nil {
		return fmt.Errorf("relay must be initialized with a call to NewRelay()")
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

	// ping every 29 seconds
	ticker := time.NewTicker(29 * time.Second)

	// to be used when the connection is closed
	go func() {
		<-r.connectionContext.Done()
		// close these things when the connection is closed
		if r.notices != nil {
			close(r.notices)
		}
		// stop the ticker
		ticker.Stop()
		// close all subscriptions
		r.Subscriptions.Range(func(_ string, sub *Subscription) bool {
			go sub.Unsub()
			return true
		})
	}()

	// queue all write operations here so we don't do mutex spaghetti
	go func() {
		for {
			select {
			case <-ticker.C:
				err := wsutil.WriteClientMessage(r.Connection.conn, ws.OpPing, nil)
				if err != nil {
					InfoLogger.Printf("{%s} error writing ping: %v; closing websocket", r.URL, err)
					r.Close() // this should trigger a context cancelation
					return
				}
			case writeRequest := <-r.writeQueue:
				// all write requests will go through this to prevent races
				if err := r.Connection.WriteMessage(writeRequest.msg); err != nil {
					writeRequest.answer <- err
				}
				close(writeRequest.answer)
			case <-r.connectionContext.Done():
				// stop here
				return
			}
		}
	}()

	// general message reader loop
	go func() {
		buf := new(bytes.Buffer)

		for {
			buf.Reset()
			if err := conn.ReadMessage(r.connectionContext, buf); err != nil {
				r.ConnectionError = err
				r.Close()
				break
			}

			message := buf.Bytes()
			debugLogf("{%s} %v\n", r.URL, message)
			envelope := ParseMessage(message)
			if envelope == nil {
				continue
			}

			switch env := envelope.(type) {
			case *NoticeEnvelope:
				// see WithNoticeHandler
				if r.notices != nil {
					r.notices <- string(*env)
				} else {
					log.Printf("NOTICE from %s: '%s'\n", r.URL, string(*env))
				}
			case *AuthEnvelope:
				if env.Challenge == nil {
					continue
				}
				r.challenge = *env.Challenge
			case *EventEnvelope:
				if env.SubscriptionID == nil {
					continue
				}
				if subscription, ok := r.Subscriptions.Load(*env.SubscriptionID); !ok {
					// InfoLogger.Printf("{%s} no subscription with id '%s'\n", r.URL, *env.SubscriptionID)
					continue
				} else {
					// check if the event matches the desired filter, ignore otherwise
					if !subscription.Filters.Match(&env.Event) {
						InfoLogger.Printf("{%s} filter does not match: %v ~ %v\n", r.URL, subscription.Filters, env.Event)
						continue
					}

					// check signature, ignore invalid, except from trusted (AssumeValid) relays
					if !r.AssumeValid {
						if ok, err := env.Event.CheckSignature(); !ok {
							errmsg := ""
							if err != nil {
								errmsg = err.Error()
							}
							InfoLogger.Printf("{%s} bad signature on %s; %s\n", r.URL, env.Event.ID, errmsg)
							continue
						}
					}

					// dispatch this to the internal .events channel of the subscription
					subscription.dispatchEvent(&env.Event)
				}
			case *EOSEEnvelope:
				if subscription, ok := r.Subscriptions.Load(string(*env)); ok {
					subscription.dispatchEose()
				}
			case *ClosedEnvelope:
				if subscription, ok := r.Subscriptions.Load(string(env.SubscriptionID)); ok {
					subscription.dispatchClosed(env.Reason)
				}
			case *CountEnvelope:
				if subscription, ok := r.Subscriptions.Load(string(env.SubscriptionID)); ok && env.Count != nil && subscription.countResult != nil {
					subscription.countResult <- *env.Count
				}
			case *OKEnvelope:
				if okCallback, exist := r.okCallbacks.Load(env.EventID); exist {
					okCallback(env.OK, env.Reason)
				} else {
					InfoLogger.Printf("{%s} got an unexpected OK message for event %s", r.URL, env.EventID)
				}
			}
		}
	}()

	return nil
}

// Write queues a message to be sent to the relay.
func (r *Relay) Write(msg []byte) <-chan error {
	ch := make(chan error)
	select {
	case r.writeQueue <- writeRequest{msg: msg, answer: ch}:
	case <-r.connectionContext.Done():
		go func() { ch <- fmt.Errorf("connection closed") }()
	}
	return ch
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
	r.okCallbacks.Store(event.ID, func(ok bool, reason string) {
		mu.Lock()
		defer mu.Unlock()
		if ok {
			status = PublishStatusSucceeded
		} else {
			status = PublishStatusFailed
			err = fmt.Errorf("msg: %s", reason)
		}
		cancel()
	})
	defer r.okCallbacks.Delete(event.ID)

	// publish event
	envb, _ := EventEnvelope{Event: event}.MarshalJSON()
	debugLogf("{%s} sending %v\n", r.URL, envb)
	status = PublishStatusSent
	if err := <-r.Write(envb); err != nil {
		status = PublishStatusFailed
		return status, err
	}

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
		}
	}
}

// Auth sends an "AUTH" command client -> relay as in NIP-42.
// Status can be: success, failed, or sent (no response from relay before ctx times out).
func (r *Relay) Auth(ctx context.Context, sign func(event *Event) error) (Status, error) {
	status := PublishStatusFailed

	authEvent := Event{
		CreatedAt: Now(),
		Kind:      KindClientAuthentication,
		Tags: Tags{
			Tag{"relay", r.URL},
			Tag{"challenge", r.challenge},
		},
		Content: "",
	}
	if err := sign(&authEvent); err != nil {
		return status, fmt.Errorf("error signing auth event: %w", err)
	}

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
	r.okCallbacks.Store(authEvent.ID, func(ok bool, reason string) {
		mu.Lock()
		if ok {
			status = PublishStatusSucceeded
		} else {
			status = PublishStatusFailed
			err = fmt.Errorf("msg: %s", reason)
		}
		mu.Unlock()
		cancel()
	})
	defer r.okCallbacks.Delete(authEvent.ID)

	// send AUTH
	authResponse, _ := AuthEnvelope{Event: authEvent}.MarshalJSON()
	debugLogf("{%s} sending %v\n", r.URL, authResponse)
	if err := <-r.Write(authResponse); err != nil {
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
//
// Remember to cancel subscriptions, either by calling `.Unsub()` on them or ensuring their `context.Context` will be canceled at some point.
// Failure to do that will result in a huge number of halted goroutines being created.
func (r *Relay) Subscribe(ctx context.Context, filters Filters, opts ...SubscriptionOption) (*Subscription, error) {
	sub := r.PrepareSubscription(ctx, filters, opts...)

	if err := sub.Fire(); err != nil {
		return nil, fmt.Errorf("couldn't subscribe to %v at %s: %w", filters, r.URL, err)
	}

	return sub, nil
}

// PrepareSubscription creates a subscription, but doesn't fire it.
//
// Remember to cancel subscriptions, either by calling `.Unsub()` on them or ensuring their `context.Context` will be canceled at some point.
// Failure to do that will result in a huge number of halted goroutines being created.
func (r *Relay) PrepareSubscription(ctx context.Context, filters Filters, opts ...SubscriptionOption) *Subscription {
	if r.Connection == nil {
		panic(fmt.Errorf("must call .Connect() first before calling .Subscribe()"))
	}

	current := subscriptionIDCounter.Add(1)
	ctx, cancel := context.WithCancel(ctx)

	sub := &Subscription{
		Relay:             r,
		Context:           ctx,
		cancel:            cancel,
		counter:           int(current),
		Events:            make(chan *Event),
		EndOfStoredEvents: make(chan struct{}),
		ClosedReason:      make(chan string, 1),
		Filters:           filters,
	}

	for _, opt := range opts {
		switch o := opt.(type) {
		case WithLabel:
			sub.label = string(o)
		}
	}

	id := sub.GetID()
	r.Subscriptions.Store(id, sub)

	// start handling events, eose, unsub etc:
	go sub.start()

	return sub
}

func (r *Relay) QuerySync(ctx context.Context, filter Filter, opts ...SubscriptionOption) ([]*Event, error) {
	sub, err := r.Subscribe(ctx, Filters{filter}, opts...)
	if err != nil {
		return nil, err
	}

	defer sub.Unsub()

	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
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

func (r *Relay) Count(ctx context.Context, filters Filters, opts ...SubscriptionOption) (int64, error) {
	sub := r.PrepareSubscription(ctx, filters, opts...)
	sub.countResult = make(chan int64)

	if err := sub.Fire(); err != nil {
		return 0, err
	}

	defer sub.Unsub()

	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 7*time.Second)
		defer cancel()
	}

	for {
		select {
		case count := <-sub.countResult:
			return count, nil
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
}

func (r *Relay) Close() error {
	r.closeMutex.Lock()
	defer r.closeMutex.Unlock()

	if r.connectionContextCancel == nil {
		return fmt.Errorf("relay not connected")
	}

	r.connectionContextCancel()
	r.connectionContextCancel = nil
	return r.Connection.Close()
}
