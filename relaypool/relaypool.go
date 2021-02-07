package relaypool

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/fiatjaf/go-nostr/event"
	nostrutils "github.com/fiatjaf/go-nostr/utils"
	"github.com/gorilla/websocket"
)

type RelayPool struct {
	SecretKey *string

	Relays     map[string]Policy
	websockets map[string]*websocket.Conn

	Events  chan *EventMessage
	Notices chan *NoticeMessage

	SubscribedKeys   []string
	SubscribedEvents []string
}

type Policy struct {
	SimplePolicy
	ReadSpecific map[string]SimplePolicy
}

type SimplePolicy struct {
	Read  bool
	Write bool
}

type EventMessage struct {
	Event   event.Event
	Context byte
	Relay   string
}

func (em *EventMessage) UnmarshalJSON(b []byte) error {
	var temp []json.RawMessage
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}
	if len(temp) < 2 {
		return errors.New("message is not an array of 2 or more")
	}
	if err := json.Unmarshal(temp[0], &em.Event); err != nil {
		return err
	}
	var context string
	if err := json.Unmarshal(temp[1], &context); err != nil {
		return err
	}
	em.Context = context[0]
	return nil
}

type NoticeMessage struct {
	Message string
	Relay   string
}

func (nm *NoticeMessage) UnmarshalJSON(b []byte) error {
	var temp []json.RawMessage
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}
	if len(temp) < 2 {
		return errors.New("message is not an array of 2 or more")
	}
	var tag string
	if err := json.Unmarshal(temp[0], &tag); err != nil {
		return err
	}
	if tag != "notice" {
		return errors.New("tag is not 'notice'")
	}

	if err := json.Unmarshal(temp[1], &nm.Message); err != nil {
		return err
	}
	return nil
}

// New creates a new RelayPool with no relays in it
func New() *RelayPool {
	return &RelayPool{
		Relays:     make(map[string]Policy),
		websockets: make(map[string]*websocket.Conn),

		Events:  make(chan *EventMessage),
		Notices: make(chan *NoticeMessage),
	}
}

// Add adds a new relay to the pool, if policy is nil, it will be a simple
// read+write policy.
func (r *RelayPool) Add(url string, policy *Policy) error {
	if policy == nil {
		policy = &Policy{SimplePolicy: SimplePolicy{Read: true, Write: true}}
	}

	nm := nostrutils.NormalizeURL(url)
	if nm == "" {
		return fmt.Errorf("invalid relay URL '%s'", url)
	}

	conn, _, err := websocket.DefaultDialer.Dial(nostrutils.WebsocketURL(url), nil)
	if err != nil {
		return fmt.Errorf("error opening websocket to '%s': %w", nm, err)
	}

	r.Relays[nm] = *policy
	r.websockets[nm] = conn

	go func() {
		for {
			typ, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read error: ", err)
				return
			}
			if typ == websocket.PingMessage {
				conn.WriteMessage(websocket.PongMessage, nil)
			}

			if typ != websocket.TextMessage || len(message) == 0 || message[0] != '[' {
				continue
			}

			var noticeMessage NoticeMessage
			var eventMessage EventMessage
			err = json.Unmarshal(message, &eventMessage)
			if err == nil {
				eventMessage.Relay = nm
				r.Events <- &eventMessage
			} else {
				err = json.Unmarshal(message, &noticeMessage)
				if err == nil {
					noticeMessage.Relay = nm
					r.Notices <- &noticeMessage
				} else {
					continue
				}
			}
		}
	}()

	return nil
}

// Remove removes a relay from the pool.
func (r *RelayPool) Remove(url string) {
	nm := nostrutils.NormalizeURL(url)
	if conn, ok := r.websockets[nm]; ok {
		conn.Close()
	}
	delete(r.Relays, nm)
	delete(r.websockets, nm)
}

func (r *RelayPool) SubKey(key string) {
	for _, conn := range r.websockets {
		conn.WriteMessage(websocket.TextMessage, []byte("sub-key:"+key))
	}
}

func (r *RelayPool) UnsubKey(key string) {
	for _, conn := range r.websockets {
		conn.WriteMessage(websocket.TextMessage, []byte("unsub-key:"+key))
	}
}

func (r *RelayPool) SubEvent(id string) {
	for _, conn := range r.websockets {
		conn.WriteMessage(websocket.TextMessage, []byte("sub-event:"+id))
	}
}

func (r *RelayPool) ReqFeed(opts map[string]interface{}) {
	var sopts string
	if opts == nil {
		sopts = "{}"
	} else {
		jopts, _ := json.Marshal(opts)
		sopts = string(jopts)
	}

	for r, conn := range r.websockets {
		err := conn.WriteMessage(websocket.TextMessage, []byte("req-feed:"+sopts))
		if err != nil {
			log.Printf("error sending req-feed to '%s': %s", r, err.Error())
		}
	}
}

func (r *RelayPool) ReqEvent(id string, opts map[string]interface{}) {
	if opts == nil {
		opts = make(map[string]interface{})
	}
	opts["id"] = id

	jopts, _ := json.Marshal(opts)
	sopts := string(jopts)

	for r, conn := range r.websockets {
		err := conn.WriteMessage(websocket.TextMessage, []byte("req-event:"+sopts))
		if err != nil {
			log.Printf("error sending req-event to '%s': %s", r, err.Error())
		}
	}
}

func (r *RelayPool) ReqKey(key string, opts map[string]interface{}) {
	if opts == nil {
		opts = make(map[string]interface{})
	}
	opts["key"] = key

	jopts, _ := json.Marshal(opts)
	sopts := string(jopts)

	for r, conn := range r.websockets {
		err := conn.WriteMessage(websocket.TextMessage, []byte("req-key:"+sopts))
		if err != nil {
			log.Printf("error sending req-key to '%s': %s", r, err.Error())
		}
	}
}

func (r *RelayPool) PublishEvent(evt *event.Event) (*event.Event, error) {
	if r.SecretKey == nil && evt.Sig == "" {
		return nil, errors.New("PublishEvent needs either a signed event to publish or to have been configured with a .SecretKey.")
	}

	if evt.Sig == "" {
		err := evt.Sign(*r.SecretKey)
		if err != nil {
			return nil, fmt.Errorf("Error signing event: %w", err)
		}
	}

	jevt, _ := json.Marshal(evt)
	for r, conn := range r.websockets {
		err := conn.WriteMessage(websocket.TextMessage, jevt)
		if err != nil {
			log.Printf("error sending event to '%s': %s", r, err.Error())
		}
	}

	return evt, nil
}
