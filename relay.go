package nostr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	s "github.com/SaveTheRbtz/generic-sync-map-go"
	"github.com/gorilla/websocket"
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

	Challenges        chan string // NIP-42 Challenges
	Notices           chan string
	ConnectionError   error
	ConnectionContext context.Context // will be canceled when the connection closes

	okCallbacks s.MapOf[string, func(bool, string)]
	mutex       sync.RWMutex

	// custom things that aren't often used
	//
	AssumeValid bool // this will skip verifying signatures for events received from this relay
}

// RelayConnect returns a relay object connected to url.
// Once successfully connected, cancelling ctx has no effect.
// To close the connection, call r.Close().
func RelayConnect(ctx context.Context, url string) (*Relay, error) {
	r := &Relay{URL: NormalizeURL(url)}
	err := r.Connect(ctx)
	return r, err
}

func (r *Relay) String() string {
	return r.URL
}

// Connect tries to establish a websocket connection to r.URL.
// If the context expires before the connection is complete, an error is returned.
// Once successfully connected, context expiration has no effect: call r.Close
// to close the connection.
func (r *Relay) Connect(ctx context.Context) error {
	connectionContext, cancel := context.WithCancel(ctx)
	r.ConnectionContext = connectionContext

	if r.URL == "" {
		cancel()
		return fmt.Errorf("invalid relay URL '%s'", r.URL)
	}

	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 7*time.Second)
		defer cancel()
	}

	socket, _, err := websocket.DefaultDialer.DialContext(ctx, r.URL, r.RequestHeader)
	if err != nil {
		cancel()
		return fmt.Errorf("error opening websocket to '%s': %w", r.URL, err)
	}

	r.Challenges = make(chan string)
	r.Notices = make(chan string)

	// close these channels when the connection is dropped
	go func() {
		<-r.ConnectionContext.Done()
		r.mutex.Lock()
		close(r.Challenges)
		close(r.Notices)
		r.mutex.Unlock()
	}()

	conn := NewConnection(socket)
	r.Connection = conn

	// ping every 29 seconds
	go func() {
		ticker := time.NewTicker(29 * time.Second)
		defer ticker.Stop()
		defer cancel()
		for {
			select {
			case <-ticker.C:
				err := conn.socket.WriteMessage(websocket.PingMessage, nil)
				if err != nil {
					InfoLogger.Printf("{%s} error writing ping: %v; closing websocket", r.URL, err)
					return
				}
			}
		}
	}()

	// handling received messages
	go func() {
		defer cancel()
		for {
			typ, message, err := conn.socket.ReadMessage()
			if err != nil {
				r.ConnectionError = err
				break
			}

			if typ == websocket.PingMessage {
				conn.WriteMessage(websocket.PongMessage, nil)
				continue
			}

			if typ != websocket.TextMessage || len(message) == 0 || message[0] != '[' {
				continue
			}

			var jsonMessage []json.RawMessage
			err = json.Unmarshal(message, &jsonMessage)
			if err != nil {
				continue
			}

			if len(jsonMessage) < 2 {
				continue
			}

			var command string
			json.Unmarshal(jsonMessage[0], &command)

			switch command {
			case "NOTICE":
				debugLog("{%s} %v\n", r.URL, jsonMessage)
				var content string
				json.Unmarshal(jsonMessage[1], &content)
				go func() {
					r.mutex.RLock()
					if r.ConnectionContext.Err() == nil {
						r.Notices <- content
					}
					r.mutex.RUnlock()
				}()
			case "AUTH":
				debugLog("{%s} %v\n", r.URL, jsonMessage)
				var challenge string
				json.Unmarshal(jsonMessage[1], &challenge)
				go func() {
					r.mutex.RLock()
					if r.ConnectionContext.Err() == nil {
						r.Challenges <- challenge
					}
					r.mutex.RUnlock()
				}()
			case "EVENT":
				if len(jsonMessage) < 3 {
					continue
				}

				var subId string
				json.Unmarshal(jsonMessage[1], &subId)
				if subscription, ok := r.subscriptions.Load(subId); !ok {
					InfoLogger.Printf("{%s} no subscription with id '%s'\n", r.URL, subId)
					continue
				} else {
					func() {
						// decode event
						var event Event
						json.Unmarshal(jsonMessage[2], &event)

						// check if the event matches the desired filter, ignore otherwise
						if !subscription.Filters.Match(&event) {
							InfoLogger.Printf("{%s} filter does not match: %v ~ %v\n", r.URL, subscription.Filters[0], event)
							return
						}

						subscription.mutex.Lock()
						defer subscription.mutex.Unlock()
						if subscription.stopped {
							return
						}

						// check signature, ignore invalid, except from trusted (AssumeValid) relays
						if !r.AssumeValid {
							if ok, err := event.CheckSignature(); !ok {
								errmsg := ""
								if err != nil {
									errmsg = err.Error()
								}
								InfoLogger.Printf("{%s} bad signature: %s\n", r.URL, errmsg)
								return
							}
						}

						subscription.Events <- &event
					}()
				}
			case "EOSE":
				if len(jsonMessage) < 2 {
					continue
				}
				debugLog("{%s} %v\n", r.URL, jsonMessage)
				var subId string
				json.Unmarshal(jsonMessage[1], &subId)
				if subscription, ok := r.subscriptions.Load(subId); ok {
					subscription.emitEose.Do(func() {
						subscription.EndOfStoredEvents <- struct{}{}
					})
				}
			case "OK":
				if len(jsonMessage) < 3 {
					continue
				}
				debugLog("{%s} %v\n", r.URL, jsonMessage)
				var (
					eventId string
					ok      bool
					msg     string
				)
				json.Unmarshal(jsonMessage[1], &eventId)
				json.Unmarshal(jsonMessage[2], &ok)

				if len(jsonMessage) > 3 {
					json.Unmarshal(jsonMessage[3], &msg)
				}

				if okCallback, exist := r.okCallbacks.Load(eventId); exist {
					okCallback(ok, msg)
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
	message := []any{"EVENT", event}
	debugLog("{%s} sending %v\n", r.URL, message)
	if err := r.Connection.WriteJSON(message); err != nil {
		return status, err
	}
	status = PublishStatusSent

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
		case <-r.ConnectionContext.Done():
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
	authResponse := []interface{}{"AUTH", event}
	debugLog("{%s} sending %v\n", r.URL, authResponse)
	if err := r.Connection.WriteJSON(authResponse); err != nil {
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
	return r.Connection.Close()
}
