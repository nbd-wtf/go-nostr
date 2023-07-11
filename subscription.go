package nostr

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
)

type Subscription struct {
	label   string
	counter int

	Relay   *Relay
	Filters Filters

	// the Events channel emits all EVENTs that come in a Subscription
	// will be closed when the subscription ends
	Events chan *Event
	events chan *Event // underlines the above, this one is never closed

	// the EndOfStoredEvents channel gets closed when an EOSE comes for that subscription
	EndOfStoredEvents chan struct{}

	// Context will be .Done() when the subscription ends
	Context context.Context

	live               atomic.Bool
	eosed              atomic.Bool
	closeEventsChannel chan struct{}
	cancel             context.CancelFunc
}

type EventMessage struct {
	Event Event
	Relay string
}

// When instantiating relay connections, some options may be passed.
// SubscriptionOption is the type of the argument passed for that.
// Some examples are WithLabel.
type SubscriptionOption interface {
	IsSubscriptionOption()
}

// WithLabel puts a label on the subscription (it is prepended to the automatic id) that is sent to relays.
type WithLabel string

func (_ WithLabel) IsSubscriptionOption() {}

var _ SubscriptionOption = (WithLabel)("")

// GetID return the Nostr subscription ID as given to the Relay
// it is a concatenation of the label and a serial number.
func (sub *Subscription) GetID() string {
	return sub.label + ":" + strconv.Itoa(sub.counter)
}

func (sub *Subscription) start() {
	var mu sync.Mutex

	for {
		select {
		case event := <-sub.events:
			// this is guarded such that it will only fire until the .Events channel is closed
			go func() {
				mu.Lock()
				if sub.live.Load() {
					sub.Events <- event
				}
				mu.Unlock()
			}()
		case <-sub.Context.Done():
			// the subscription ends once the context is canceled
			sub.Unsub()
			return
		case <-sub.closeEventsChannel:
			// this is called only once on .Unsub() and closes the .Events channel
			mu.Lock()
			close(sub.Events)
			mu.Unlock()
			return
		}
	}
}

// Unsub closes the subscription, sending "CLOSE" to relay as in NIP-01.
// Unsub() also closes the channel sub.Events and makes a new one.
func (sub *Subscription) Unsub() {
	sub.cancel()

	// naïve sync.Once implementation:
	if sub.live.CompareAndSwap(true, false) {
		go sub.Close()
		id := sub.GetID()
		sub.Relay.Subscriptions.Delete(id)

		// do this so we don't have the possibility of closing the Events channel and then trying to send to it
		close(sub.closeEventsChannel)
	}
}

// Close just sends a CLOSE message. You probably want Unsub() instead.
func (sub *Subscription) Close() {
	if sub.Relay.IsConnected() {
		id := sub.GetID()
		closeMsg := CloseEnvelope(id)
		closeb, _ := (&closeMsg).MarshalJSON()
		debugLogf("{%s} sending %v", sub.Relay.URL, closeb)
		<-sub.Relay.Write(closeb)
	}
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

	reqb, _ := ReqEnvelope{id, sub.Filters}.MarshalJSON()
	debugLogf("{%s} sending %v", sub.Relay.URL, reqb)

	sub.live.Store(true)
	if err := <-sub.Relay.Write(reqb); err != nil {
		sub.cancel()
		return fmt.Errorf("failed to write: %w", err)
	}

	return nil
}
