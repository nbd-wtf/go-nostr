package nostr

import (
	"context"
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

// Add calls AddContext with background context in a separate goroutine, sending
// any connection error over the returned channel.
//
// The returned channel is closed once the connection is successfully
// established or RelayConnectContext returned an error.
func (r *RelayPool) Add(url string, policy RelayPoolPolicy) <-chan error {
	cherr := make(chan error)
	go func() {
		defer close(cherr)
		if err := r.AddContext(context.Background(), url, policy); err != nil {
			cherr <- err
		}
	}()
	return cherr
}

// AddContext connects to a relay at a canonical version specified by the url
// and adds it to the pool. The returned error is non-nil only on connection
// errors, including an expired context before the connection is complete.
//
// Once successfully connected, AddContext returns and the context expiration
// has no effect: call r.Remove to close the connection and delete a relay from the pool.
func (r *RelayPool) AddContext(ctx context.Context, url string, policy RelayPoolPolicy) error {
	relay, err := RelayConnectContext(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", url, err)
	}
	if policy == nil {
		policy = SimplePolicy{Read: true, Write: true}
	}
	r.addConnected(relay, policy)
	return nil
}

func (r *RelayPool) addConnected(relay *Relay, policy RelayPoolPolicy) {
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

//Sub subscribes to events matching the passed filters and returns the subscription ID,
//a channel which you should pass into Unique to get unique events, and a function which
//you should call to clean up and close your subscription so that the relay doesn't block you.
func (r *RelayPool) Sub(filters Filters) (subID string, events chan EventMessage, unsubscribe func()) {
	random := make([]byte, 7)
	rand.Read(random)
	id := hex.EncodeToString(random)

	r.subscriptions.Store(id, filters)
	eventStream := make(chan EventMessage)
	r.eventStreams.Store(id, eventStream)
	unsub := make(chan struct{})

	r.Relays.Range(func(_ string, relay *Relay) bool {
		sub := relay.prepareSubscription(id)
		sub.Sub(filters)

		go func(sub *Subscription) {
			for evt := range sub.Events {
				eventStream <- EventMessage{Relay: relay.URL, Event: evt}
			}
		}(sub)

		go func() {
			select {
			case <-unsub:
				sub.Unsub()
			}
		}()

		return true
	})

	return id, eventStream, func() { gracefulClose(unsub) }
}

func gracefulClose(c chan struct{}) {
	select {
	case <-c:
	default:
		close(c)
	}
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
