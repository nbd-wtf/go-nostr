package nostr

import (
	"context"
	"fmt"
	"strconv"
	"sync"
)

type Subscription struct {
	label   string
	counter int
	conn    *Connection
	mutex   sync.Mutex

	Relay             *Relay
	Filters           Filters
	Events            chan *Event
	EndOfStoredEvents chan struct{}
	Context           context.Context
	cancel            context.CancelFunc

	stopped  bool
	emitEose sync.Once
}

type EventMessage struct {
	Event Event
	Relay string
}

// SetLabel puts a label on the subscription that is prepended to the id that is sent to relays,
//
//	it's only useful for debugging and sanity purposes.
func (sub *Subscription) SetLabel(label string) {
	sub.label = label
}

// GetID return the Nostr subscription ID as given to the relay, it will be a sequential number, stringified.
func (sub *Subscription) GetID() string {
	return sub.label + ":" + strconv.Itoa(sub.counter)
}

// Unsub closes the subscription, sending "CLOSE" to relay as in NIP-01.
// Unsub() also closes the channel sub.Events.
func (sub *Subscription) Unsub() {
	sub.mutex.Lock()
	defer sub.mutex.Unlock()

	id := sub.GetID()
	closeMsg := CloseEnvelope(id)
	closeb, _ := (&closeMsg).MarshalJSON()
	debugLog("{%s} sending %v", sub.Relay.URL, closeb)
	sub.conn.WriteMessage(closeb)
	sub.Relay.Subscriptions.Delete(id)

	if !sub.stopped && sub.Events != nil {
		close(sub.Events)
	}
	sub.stopped = true
}

// Sub sets sub.Filters and then calls sub.Fire(ctx).
// The subscription will be closed if the context expires.
func (sub *Subscription) Sub(ctx context.Context, filters Filters) {
	sub.Filters = filters
	sub.Fire()
}

// Fire sends the "REQ" command to the relay.
func (sub *Subscription) Fire() error {
	id := sub.GetID()
	sub.Relay.Subscriptions.Store(id, sub)

	reqb, _ := ReqEnvelope{id, sub.Filters}.MarshalJSON()
	debugLog("{%s} sending %v", sub.Relay.URL, reqb)
	if err := sub.conn.WriteMessage(reqb); err != nil {
		sub.cancel()
		return fmt.Errorf("failed to write: %w", err)
	}

	// the subscription ends once the context is canceled
	go func() {
		<-sub.Context.Done()
		sub.Unsub()
	}()

	// or when the relay connection is closed
	go func() {
		<-sub.Relay.connectionContext.Done()

		// cancel the context -- this will cause the other context cancelation cause above to be called
		sub.cancel()
	}()

	return nil
}
