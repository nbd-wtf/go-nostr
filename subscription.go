package nostr

import (
	s "github.com/SaveTheRbtz/generic-sync-map-go"
)

type Subscription struct {
	channel string
	relays  s.MapOf[string, *Connection]

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
	subscription.relays.Range(func(_ string, conn *Connection) bool {
		conn.WriteJSON([]interface{}{
			"CLOSE",
			subscription.channel,
		})
		return true
	})

	subscription.stopped = true
	if subscription.Events != nil {
		close(subscription.Events)
	}
	if subscription.UniqueEvents != nil {
		close(subscription.UniqueEvents)
	}
}

func (subscription *Subscription) Sub(filters Filters) {
	subscription.filters = filters

	subscription.relays.Range(func(_ string, conn *Connection) bool {
		message := []interface{}{
			"REQ",
			subscription.channel,
		}
		for _, filter := range subscription.filters {
			message = append(message, filter)
		}

		conn.WriteJSON(message)
		return true
	})

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
	if conn, ok := subscription.relays.Load(relay); ok {
		subscription.relays.Delete(relay)
		conn.WriteJSON([]interface{}{
			"CLOSE",
			subscription.channel,
		})
	}
}

func (subscription Subscription) addRelay(relay string, conn *Connection) {
	subscription.relays.Store(relay, conn)

	message := []interface{}{
		"REQ",
		subscription.channel,
	}
	for _, filter := range subscription.filters {
		message = append(message, filter)
	}

	conn.WriteJSON(message)
}
