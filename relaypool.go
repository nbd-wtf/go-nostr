package nostr

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	s "github.com/SaveTheRbtz/generic-sync-map-go"
)

type PublishStatus struct {
	Relay  string
	Status Status
}

type RelayPool struct {
	SecretKey *string

	Policies      s.MapOf[string, RelayPoolPolicy]
	Relays        s.MapOf[string, *Relay]
	subscriptions s.MapOf[string, Filters]
	eventStreams  s.MapOf[string, chan EventMessage]

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
		Policies: s.MapOf[string, RelayPoolPolicy]{},
		Relays:   s.MapOf[string, *Relay]{},

		Notices: make(chan *NoticeMessage),
	}
}

// Add adds a new relay to the pool, if policy is nil, it will be a simple
// read+write policy.
func (r *RelayPool) Add(url string, policy RelayPoolPolicy) chan error {
	if policy == nil {
		policy = SimplePolicy{Read: true, Write: true}
	}

	cherr := make(chan error)

	go func() {
		relay, err := RelayConnect(url)
		if err != nil {
			cherr <- fmt.Errorf("failed to connect to %s: %w", url, err)
			return
		}

		r.Policies.Store(relay.URL, policy)
		r.Relays.Store(relay.URL, relay)

		r.subscriptions.Range(func(id string, filters Filters) bool {
			sub := relay.prepareSubscription(id)
			sub.Sub(filters)
			eventStream, _ := r.eventStreams.Load(id)

			go func(sub *Subscription) {
				for evt := range sub.Events {
					eventStream <- EventMessage{Relay: relay.URL, Event: evt}
				}
			}(sub)

			return true
		})

		cherr <- nil
		close(cherr)
	}()

	return cherr
}

// Remove removes a relay from the pool.
func (r *RelayPool) Remove(url string) {
	nm := NormalizeURL(url)

	r.Relays.Delete(nm)
	r.Policies.Delete(nm)

	if relay, ok := r.Relays.Load(nm); ok {
		relay.Close()
	}
}

func (r *RelayPool) Sub(filters Filters) (string, chan EventMessage) {
	random := make([]byte, 7)
	rand.Read(random)
	id := hex.EncodeToString(random)

	r.subscriptions.Store(id, filters)
	eventStream := make(chan EventMessage)
	r.eventStreams.Store(id, eventStream)

	r.Relays.Range(func(_ string, relay *Relay) bool {
		sub := relay.prepareSubscription(id)
		sub.Sub(filters)

		go func(sub *Subscription) {
			for evt := range sub.Events {
				eventStream <- EventMessage{Relay: relay.URL, Event: evt}
			}
		}(sub)

		return true
	})

	return id, eventStream
}

func Unique(all chan EventMessage) chan Event {
	uniqueEvents := make(chan Event)
	emittedAlready := s.MapOf[string, struct{}]{}

	go func() {
		for eventMessage := range all {
			if _, ok := emittedAlready.LoadOrStore(eventMessage.Event.ID, struct{}{}); !ok {
				uniqueEvents <- eventMessage.Event
			}
		}
	}()

	return uniqueEvents
}

func (r *RelayPool) PublishEvent(evt *Event) (*Event, chan PublishStatus, error) {
	size := 0
	r.Relays.Range(func(_ string, _ *Relay) bool {
		size++
		return true
	})
	status := make(chan PublishStatus, size)

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

	r.Relays.Range(func(url string, relay *Relay) bool {
		if r, ok := r.Policies.Load(url); !ok || !r.ShouldWrite(evt) {
			return true
		}

		go func(relay *Relay) {
			for resultStatus := range relay.Publish(*evt) {
				status <- PublishStatus{relay.URL, resultStatus}
			}
		}(relay)

		return true
	})

	return evt, status, nil
}
