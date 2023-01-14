package nostr

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
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
	URL string

	Connection    *Connection
	subscriptions s.MapOf[string, *Subscription]

	Notices         chan string
	ConnectionError chan error

	okCallbacks s.MapOf[string, func(bool)]
}

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
	if r.URL == "" {
		return fmt.Errorf("invalid relay URL '%s'", r.URL)
	}

	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 7 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 7*time.Second)
		defer cancel()
	}

	socket, _, err := websocket.DefaultDialer.DialContext(ctx, r.URL, nil)
	if err != nil {
		return fmt.Errorf("error opening websocket to '%s': %w", r.URL, err)
	}

	r.Notices = make(chan string)
	r.ConnectionError = make(chan error)

	conn := NewConnection(socket)
	r.Connection = conn

	go func() {
		for {
			typ, message, err := conn.socket.ReadMessage()
			if err != nil {
				r.ConnectionError <- err
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

			var label string
			json.Unmarshal(jsonMessage[0], &label)

			switch label {
			case "NOTICE":
				var content string
				json.Unmarshal(jsonMessage[1], &content)
				r.Notices <- content
			case "EVENT":
				if len(jsonMessage) < 3 {
					continue
				}

				var channel string
				json.Unmarshal(jsonMessage[1], &channel)
				if subscription, ok := r.subscriptions.Load(channel); ok {
					var event Event
					json.Unmarshal(jsonMessage[2], &event)

					// check signature of all received events, ignore invalid
					ok, err := event.CheckSignature()
					if !ok {
						errmsg := ""
						if err != nil {
							errmsg = err.Error()
						}
						log.Printf("bad signature: %s", errmsg)
						continue
					}

					// check if the event matches the desired filter, ignore otherwise
					if !subscription.Filters.Match(&event) {
						continue
					}

					if !subscription.stopped {
						subscription.Events <- event
					}
				}
			case "EOSE":
				if len(jsonMessage) < 2 {
					continue
				}
				var channel string
				json.Unmarshal(jsonMessage[1], &channel)
				if subscription, ok := r.subscriptions.Load(channel); ok {
					subscription.emitEose.Do(func() {
						subscription.EndOfStoredEvents <- struct{}{}
					})
				}
			case "OK":
				if len(jsonMessage) < 3 {
					continue
				}
				var (
					eventId string
					ok      bool
					msg     string
				)
				json.Unmarshal(jsonMessage[1], &eventId)
				json.Unmarshal(jsonMessage[2], &ok)
				json.Unmarshal(jsonMessage[3], &msg)

				log.Println(msg)

				if okCallback, exist := r.okCallbacks.Load(eventId); exist {
					okCallback(ok)
				}
			}
		}
	}()

	return nil
}

func (r *Relay) Publish(ctx context.Context, event Event) Status {
	status := PublishStatusFailed

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
	okCallback := func(ok bool) {
		if ok {
			status = PublishStatusSucceeded
		} else {
			status = PublishStatusFailed
		}
		cancel()
	}
	r.okCallbacks.Store(event.ID, okCallback)
	defer r.okCallbacks.Delete(event.ID)

	// publish event
	err := r.Connection.WriteJSON([]interface{}{"EVENT", event})
	if err != nil {
		return status
	}

	// update status (this will be returned later)
	status = PublishStatusSent

	sub := r.Subscribe(ctx, Filters{Filter{IDs: []string{event.ID}}})
	for {
		select {
		case receivedEvent := <-sub.Events:
			if receivedEvent.ID == event.ID {
				// we got a success, so update our status and proceed to return
				status = PublishStatusSucceeded
				return status
			}
		case <-ctx.Done():
			// return status as it was
			// will proceed to return status as it is
			// e.g. if this happens because of the timeout then status will probably be "failed"
			//      but if it happens because okCallback was called then it might be "succeeded"
			return status
		}
	}
}

func (r *Relay) Subscribe(ctx context.Context, filters Filters) *Subscription {
	if r.Connection == nil {
		panic(fmt.Errorf("must call .Connect() first before calling .Subscribe()"))
	}

	sub := r.PrepareSubscription()
	sub.Filters = filters
	sub.Fire(ctx)

	return sub
}

func (r *Relay) QuerySync(ctx context.Context, filter Filter) []Event {
	sub := r.Subscribe(ctx, Filters{filter})
	defer sub.Unsub()

	if _, ok := ctx.Deadline(); !ok {
		// if no timeout is set, force it to 3 seconds
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}

	var events []Event
	for {
		select {
		case evt := <-sub.Events:
			events = append(events, evt)
		case <-sub.EndOfStoredEvents:
			return events
		case <-ctx.Done():
			return events
		}
	}
}

func (r *Relay) PrepareSubscription() *Subscription {
	random := make([]byte, 7)
	rand.Read(random)
	id := hex.EncodeToString(random)

	return r.prepareSubscription(id)
}

func (r *Relay) prepareSubscription(id string) *Subscription {
	sub := &Subscription{
		Relay:             r,
		conn:              r.Connection,
		id:                id,
		Events:            make(chan Event),
		EndOfStoredEvents: make(chan struct{}, 1),
	}

	r.subscriptions.Store(sub.id, sub)
	return sub
}

func (r *Relay) Close() error {
	return r.Connection.Close()
}
