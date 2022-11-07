package nostr

type Subscription struct {
	id   string
	conn *Connection

	filters Filters
	Events  chan Event

	stopped bool
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
	sub.filters = filters

	message := []interface{}{"REQ", sub.id}
	for _, filter := range sub.filters {
		message = append(message, filter)
	}

	sub.conn.WriteJSON(message)
}
