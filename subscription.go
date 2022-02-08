package nostr

type Subscription struct {
	channel string
	relays  map[string]*Connection

	filters Filters
	Events  chan EventMessage

	started      bool
	UniqueEvents chan Event

	stopped bool
}

type EventMessage struct {
	Event Event
	Relay string
}

func (subscription Subscription) Unsub() {
	for _, conn := range subscription.relays {
		conn.WriteJSON([]interface{}{
			"CLOSE",
			subscription.channel,
		})
	}

	subscription.stopped = true
	if subscription.Events != nil {
		close(subscription.Events)
	}
	if subscription.UniqueEvents != nil {
		close(subscription.UniqueEvents)
	}
}

func (subscription Subscription) Sub() {
	for _, conn := range subscription.relays {
		message := []interface{}{
			"REQ",
			subscription.channel,
		}
		for _, filter := range subscription.filters {
			message = append(message, filter)
		}

		conn.WriteJSON(message)
	}

	if !subscription.started {
		go subscription.startHandlingUnique()
	}
}

func (subscription Subscription) startHandlingUnique() {
	seen := make(map[string]struct{})
	for em := range subscription.Events {
		if _, ok := seen[em.Event.ID]; ok {
			continue
		}
		seen[em.Event.ID] = struct{}{}
		if !subscription.stopped {
			subscription.UniqueEvents <- em.Event
		}
	}
}

func (subscription Subscription) removeRelay(relay string) {
	if conn, ok := subscription.relays[relay]; ok {
		delete(subscription.relays, relay)
		conn.WriteJSON([]interface{}{
			"CLOSE",
			subscription.channel,
		})
	}
}

func (subscription Subscription) addRelay(relay string, conn *Connection) {
	subscription.relays[relay] = conn

	message := []interface{}{
		"REQ",
		subscription.channel,
	}
	for _, filter := range subscription.filters {
		message = append(message, filter)
	}

	conn.WriteJSON(message)
}
