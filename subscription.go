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
//   it's only useful for debugging and sanity purposes.
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

	message := []any{"CLOSE", sub.GetID()}
	debugLog("{%s} sending %v", sub.Relay.URL, message)
	sub.conn.WriteJSON(message)

	if sub.stopped == false && sub.Events != nil {
		close(sub.Events)
	}
	sub.stopped = true
}

// Sub sets sub.Filters and then calls sub.Fire(ctx).
func (sub *Subscription) Sub(ctx context.Context, filters Filters) {
	sub.Filters = filters
	sub.Fire()
}

// Fire sends the "REQ" command to the relay.
func (sub *Subscription) Fire() error {
	sub.Relay.subscriptions.Store(sub.GetID(), sub)

	message := []interface{}{"REQ", sub.GetID()}
	for _, filter := range sub.Filters {
		message = append(message, filter)
	}

	debugLog("{%s} sending %v", sub.Relay.URL, message)

	err := sub.conn.WriteJSON(message)
	if err != nil {
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
		<-sub.Relay.ConnectionContext.Done()

		// cancel the context -- this will cause the other context cancelation cause above to be called
		sub.cancel()
	}()

	return nil
}
