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

	Relay   *Relay
	Filters Filters

	// the Events channel emits all EVENTs that come in a Subscription
	// will be closed when the subscription ends
	Events chan *Event

	// the EndOfStoredEvents channel gets closed when an EOSE comes for that subscription
	EndOfStoredEvents chan struct{}

	// Context will be .Done() when the subscription ends
	Context context.Context

	live     bool
	cancel   context.CancelFunc
	emitEose sync.Once
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

// Unsub closes the subscription, sending "CLOSE" to relay as in NIP-01.
// Unsub() also closes the channel sub.Events and makes a new one.
func (sub *Subscription) Unsub() {
	id := sub.GetID()

	if sub.Relay.IsConnected() {
		closeMsg := CloseEnvelope(id)
		closeb, _ := (&closeMsg).MarshalJSON()
		debugLog("{%s} sending %v", sub.Relay.URL, closeb)
		sub.Relay.Write(closeb)
	}

	sub.live = false
	sub.Relay.Subscriptions.Delete(id)

	// do this so we don't have the possibility of closing the Events channel and then trying to send to it
	sub.Relay.subscriptionChannelCloseQueue <- sub
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
	debugLog("{%s} sending %v", sub.Relay.URL, reqb)

	sub.live = true
	if err := sub.Relay.Write(reqb); err != nil {
		sub.cancel()
		return fmt.Errorf("failed to write: %w", err)
	}

	return nil
}
