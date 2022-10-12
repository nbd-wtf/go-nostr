package nostr

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
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

type PublishStatus struct {
	Relay  string
	Status Status
}

type RelayPool struct {
	SecretKey *string

	Relays        s.MapOf[string, RelayPoolPolicy]
	websockets    s.MapOf[string, *Connection]
	subscriptions s.MapOf[string, *Subscription]

	Notices chan *NoticeMessage
}

type RelayPoolPolicy interface {
	ShouldRead(Filters) bool
	ShouldWrite(*Event) bool
}

type SimplePolicy struct {
	Read  bool
	Write bool
}

func (s SimplePolicy) ShouldRead(_ Filters) bool {
	return s.Read
}

func (s SimplePolicy) ShouldWrite(_ *Event) bool {
	return s.Write
}

type NoticeMessage struct {
	Message string
	Relay   string
}

// New creates a new RelayPool with no relays in it
func NewRelayPool() *RelayPool {
	return &RelayPool{
		Relays:        s.MapOf[string, RelayPoolPolicy]{},
		websockets:    s.MapOf[string, *Connection]{},
		subscriptions: s.MapOf[string, *Subscription]{},

		Notices: make(chan *NoticeMessage),
	}
}

// Add adds a new relay to the pool, if policy is nil, it will be a simple
// read+write policy.
func (r *RelayPool) Add(url string, policy RelayPoolPolicy) error {
	if policy == nil {
		policy = SimplePolicy{Read: true, Write: true}
	}

	nm := NormalizeURL(url)
	if nm == "" {
		return fmt.Errorf("invalid relay URL '%s'", url)
	}

	socket, _, err := websocket.DefaultDialer.Dial(NormalizeURL(url), nil)
	if err != nil {
		return fmt.Errorf("error opening websocket to '%s': %w", nm, err)
	}

	conn := NewConnection(socket)

	r.Relays.Store(nm, policy)
	r.websockets.Store(nm, conn)

	r.subscriptions.Range(func(_ string, sub *Subscription) bool {
		sub.addRelay(nm, conn)
		return true
	})

	go func() {
		for {
			typ, message, err := conn.socket.ReadMessage()
			if err != nil {
				log.Println("read error: ", err)
				return
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
				r.Notices <- &NoticeMessage{
					Relay:   nm,
					Message: content,
				}
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
					if !subscription.filters.Match(&event) {
						continue
					}

					if !subscription.stopped {
						subscription.Events <- EventMessage{
							Relay: nm,
							Event: event,
						}
					}
				}
			}
		}
	}()

	return nil
}

// Remove removes a relay from the pool.
func (r *RelayPool) Remove(url string) {
	nm := NormalizeURL(url)

	r.subscriptions.Range(func(_ string, sub *Subscription) bool {
		sub.removeRelay(nm)
		return true
	})

	if conn, ok := r.websockets.Load(nm); ok {
		conn.Close()
	}

	r.Relays.Delete(nm)
	r.websockets.Delete(nm)
}

func (r *RelayPool) Sub(filters Filters) *Subscription {
	random := make([]byte, 7)
	rand.Read(random)

	subscription := Subscription{}
	subscription.channel = hex.EncodeToString(random)
	subscription.relays = s.MapOf[string, *Connection]{}

	r.Relays.Range(func(relay string, policy RelayPoolPolicy) bool {
		if policy.ShouldRead(filters) {
			if ws, ok := r.websockets.Load(relay); ok {
				subscription.relays.Store(relay, ws)
			}
		}
		return true
	})
	subscription.Events = make(chan EventMessage)
	subscription.UniqueEvents = make(chan Event)
	r.subscriptions.Store(subscription.channel, &subscription)

	subscription.Sub(filters)
	return &subscription
}

func (r *RelayPool) PublishEvent(evt *Event) (*Event, chan PublishStatus, error) {
	status := make(chan PublishStatus, 1)

	if r.SecretKey == nil && (evt.PubKey == "" || evt.Sig == "") {
		return nil, status, errors.New("PublishEvent needs either a signed event to publish or to have been configured with a .SecretKey.")
	}

	if evt.PubKey == "" {
		sk, err := GetPublicKey(*r.SecretKey)
		if err != nil {
			return nil, status, fmt.Errorf("The pool's global SecretKey is invalid: %w", err)
		}
		evt.PubKey = sk
	}

	if evt.Sig == "" {
		err := evt.Sign(*r.SecretKey)
		if err != nil {
			return nil, status, fmt.Errorf("Error signing event: %w", err)
		}
	}

	r.websockets.Range(func(relay string, conn *Connection) bool {
		if r, ok := r.Relays.Load(relay); !ok || !r.ShouldWrite(evt) {
			return true
		}

		go func(relay string, conn *Connection) {
			err := conn.WriteJSON([]interface{}{"EVENT", evt})
			if err != nil {
				log.Printf("error sending event to '%s': %s", relay, err.Error())
				status <- PublishStatus{relay, PublishStatusFailed}
			}
			status <- PublishStatus{relay, PublishStatusSent}

			subscription := r.Sub(Filters{Filter{IDs: []string{evt.ID}}})
			for {
				select {
				case event := <-subscription.UniqueEvents:
					if event.ID == evt.ID {
						status <- PublishStatus{relay, PublishStatusSucceeded}
						break
					} else {
						continue
					}
				case <-time.After(5 * time.Second):
					break
				}
				break
			}
		}(relay, conn)

		return true
	})

	return evt, status, nil
}
