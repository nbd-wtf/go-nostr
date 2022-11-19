package nostr

import "sync"

type Subscription struct {
	id   string
	conn *Connection

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

func (sub Subscription) Unsub() {
	sub.conn.WriteJSON([]interface{}{"CLOSE", sub.id})

	sub.stopped = true
	if sub.Events != nil {
		close(sub.Events)
	}
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
