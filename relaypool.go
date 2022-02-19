package nostr

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/fiatjaf/bip340"
	"github.com/gorilla/websocket"
)

const (
	PublishStatusSent      = 0
	PublishStatusFailed    = -1
	PublishStatusSucceeded = 1
)

type PublishStatus struct {
	Relay  string
	Status int
}

type RelayPool struct {
	SecretKey *string

	Relays        map[string]RelayPoolPolicy
	websockets    map[string]*Connection
	subscriptions map[string]*Subscription

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
		Relays:        make(map[string]RelayPoolPolicy),
		websockets:    make(map[string]*Connection),
		subscriptions: make(map[string]*Subscription),

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

	r.Relays[nm] = policy
	r.websockets[nm] = conn

	for _, sub := range r.subscriptions {
		sub.addRelay(nm, conn)
	}

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
				if subscription, ok := r.subscriptions[channel]; ok {
					var event Event
					json.Unmarshal(jsonMessage[2], &event)

					// check signature of all received events, ignore invalid
					ok, _ := event.CheckSignature()
					if !ok {
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

	for _, sub := range r.subscriptions {
		sub.removeRelay(nm)
	}
	if conn, ok := r.websockets[nm]; ok {
		conn.Close()
	}

	delete(r.Relays, nm)
	delete(r.websockets, nm)
}

func (r *RelayPool) Sub(filters Filters) *Subscription {
	random := make([]byte, 7)
	rand.Read(random)

	subscription := Subscription{filters: filters}
	subscription.channel = hex.EncodeToString(random)
	subscription.relays = make(map[string]*Connection)
	for relay, policy := range r.Relays {
		if policy.ShouldRead(filters) {
			ws := r.websockets[relay]
			subscription.relays[relay] = ws
		}
	}
	subscription.Events = make(chan EventMessage)
	subscription.UniqueEvents = make(chan Event)
	r.subscriptions[subscription.channel] = &subscription

	subscription.Sub()
	return &subscription
}

func (r *RelayPool) PublishEvent(evt *Event) (*Event, chan PublishStatus, error) {
	status := make(chan PublishStatus, 1)

	if r.SecretKey == nil && (evt.PubKey == "" || evt.Sig == "") {
		return nil, status, errors.New("PublishEvent needs either a signed event to publish or to have been configured with a .SecretKey.")
	}

	if evt.PubKey == "" {
		secretKeyN, err := bip340.ParsePrivateKey(*r.SecretKey)
		if err != nil {
			return nil, status, fmt.Errorf("The pool's global SecretKey is invalid: %w", err)
		}
		evt.PubKey = fmt.Sprintf("%x", bip340.GetPublicKey(secretKeyN))
	}

	if evt.Sig == "" {
		err := evt.Sign(*r.SecretKey)
		if err != nil {
			return nil, status, fmt.Errorf("Error signing event: %w", err)
		}
	}

	for relay, conn := range r.websockets {
		if !r.Relays[relay].ShouldWrite(evt) {
			continue
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
			subscription.Unsub()
			close(status)
		}(relay, conn)
	}

	return evt, status, nil
}
