package nostr

import (
	"sync"
)

type Subscription struct {
	id    string
	conn  *Connection
	mutex sync.Mutex

	Relay             *Relay
	Filters           Filters
	Events            chan Event
	EndOfStoredEvents chan struct{}

	stopped  bool
	emitEose sync.Once
}

type EventMessage struct {
	Event Event
	Relay string
}

func (sub *Subscription) Unsub() {
	sub.mutex.Lock()
	defer sub.mutex.Unlock()

	sub.conn.WriteJSON([]interface{}{"CLOSE", sub.id})
	if sub.stopped == false && sub.Events != nil {
		close(sub.Events)
	}
	sub.stopped = true
}

func (sub *Subscription) Sub(filters Filters) {
	sub.Filters = filters
	sub.Fire()
}

func (sub *Subscription) Fire() {
	message := []interface{}{"REQ", sub.id}
	for _, filter := range sub.Filters {
		message = append(message, filter)
	}

	sub.conn.WriteJSON(message)
}
