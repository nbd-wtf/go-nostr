package nostr

import (
	"context"
	"strconv"
	"sync"
)

type Subscription struct {
	label string
	id    int
	conn  *Connection
	mutex sync.Mutex

	Relay             *Relay
	Filters           Filters
	Events            chan *Event
	EndOfStoredEvents chan struct{}
	Context           context.Context

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
	return sub.label + ":" + strconv.Itoa(sub.id)
}

// Unsub closes the subscription, sending "CLOSE" to relay as in NIP-01.
// Unsub() also closes the channel sub.Events.
func (sub *Subscription) Unsub() {
	sub.mutex.Lock()
	defer sub.mutex.Unlock()

	sub.conn.WriteJSON([]interface{}{"CLOSE", sub.GetID()})
	if sub.stopped == false && sub.Events != nil {
		close(sub.Events)
	}
	sub.stopped = true
}

// Sub sets sub.Filters and then calls sub.Fire(ctx).
func (sub *Subscription) Sub(ctx context.Context, filters Filters) {
	sub.Filters = filters
	sub.Fire(ctx)
}

// Fire sends the "REQ" command to the relay.
// When ctx is cancelled, sub.Unsub() is called, closing the subscription.
func (sub *Subscription) Fire(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	sub.Context = ctx

	message := []interface{}{"REQ", sub.GetID()}
	for _, filter := range sub.Filters {
		message = append(message, filter)
	}

	err := sub.conn.WriteJSON(message)
	if err != nil {
		cancel()
		return err
	}

	// the subscription ends once the context is canceled
	go func() {
		<-ctx.Done()
		sub.Unsub()
	}()

	// or when the relay connection is closed
	go func() {
		<-sub.Relay.ConnectionContext.Done()

		// this will close the Events channel,
		// which can be used by an external reader to learn the subscription has stopped
		sub.Unsub()

		// we also cancel the context
		cancel()
	}()

	return nil
}
