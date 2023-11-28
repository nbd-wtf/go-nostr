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

	// for this to be treated as a COUNT and not a REQ this must be set
	countResult chan int64

	// the Events channel emits all EVENTs that come in a Subscription
	// will be closed when the subscription ends
	Events chan *Event
	mu     sync.Mutex

	// the EndOfStoredEvents channel gets closed when an EOSE comes for that subscription
	EndOfStoredEvents chan struct{}

	// the ClosedReason channel emits the reason when a CLOSED message is received
	ClosedReason chan string

	// Context will be .Done() when the subscription ends
	Context context.Context

	live   atomic.Bool
	eosed  atomic.Bool
	closed atomic.Bool
	cancel context.CancelFunc

	// this keeps track of the events we've received before the EOSE that we must dispatch before
	// closing the EndOfStoredEvents channel
	storedwg sync.WaitGroup
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
	<-sub.Context.Done()
	// the subscription ends once the context is canceled (if not already)
	sub.Unsub() // this will set sub.live to false

	// do this so we don't have the possibility of closing the Events channel and then trying to send to it
	sub.mu.Lock()
	close(sub.Events)
	sub.mu.Unlock()
}

func (sub *Subscription) dispatchEvent(evt *Event) {
	added := false
	if !sub.eosed.Load() {
		sub.storedwg.Add(1)
		added = true
	}

	go func() {
		sub.mu.Lock()
		defer sub.mu.Unlock()

		if sub.live.Load() {
			select {
			case sub.Events <- evt:
			case <-sub.Context.Done():
			}
		}

		if added {
			sub.storedwg.Done()
		}
	}()
}

func (sub *Subscription) dispatchEose() {
	if sub.eosed.CompareAndSwap(false, true) {
		go func() {
			sub.storedwg.Wait()
			close(sub.EndOfStoredEvents)
		}()
	}
}

func (sub *Subscription) dispatchClosed(reason string) {
	if sub.closed.CompareAndSwap(false, true) {
		sub.ClosedReason <- reason
		close(sub.ClosedReason)
	}
}

// Unsub closes the subscription, sending "CLOSE" to relay as in NIP-01.
// Unsub() also closes the channel sub.Events and makes a new one.
func (sub *Subscription) Unsub() {
	// cancel the context (if it's not canceled already)
	sub.cancel()

	// mark subscription as closed and send a CLOSE to the relay (naïve sync.Once implementation)
	if sub.live.CompareAndSwap(true, false) {
		sub.Close()
	}

	// remove subscription from our map
	sub.Relay.Subscriptions.Delete(sub.GetID())
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
func (sub *Subscription) Sub(_ context.Context, filters Filters) {
	sub.Filters = filters
	sub.Fire()
}

// Fire sends the "REQ" command to the relay.
func (sub *Subscription) Fire() error {
	id := sub.GetID()

	var reqb []byte
	if sub.countResult == nil {
		reqb, _ = ReqEnvelope{id, sub.Filters}.MarshalJSON()
	} else {
		reqb, _ = CountEnvelope{id, sub.Filters, nil}.MarshalJSON()
	}
	debugLogf("{%s} sending %v", sub.Relay.URL, reqb)

	sub.live.Store(true)
	if err := <-sub.Relay.Write(reqb); err != nil {
		sub.cancel()
		return fmt.Errorf("failed to write: %w", err)
	}

	return nil
}
