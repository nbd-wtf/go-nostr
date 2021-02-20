package relaypool

import (
	"github.com/fiatjaf/go-nostr/event"
	"github.com/fiatjaf/go-nostr/filter"
	"github.com/gorilla/websocket"
)

type Subscription struct {
	channel string
	relays  map[string]*websocket.Conn

	filter *filter.EventFilter
	Events chan EventMessage

	started      bool
	UniqueEvents chan event.Event
}

type EventMessage struct {
	Event event.Event
	Relay string
}

func (subscription Subscription) Unsub() {
	for _, ws := range subscription.relays {
		ws.WriteJSON([]interface{}{
			"CLOSE",
			subscription.channel,
		})
	}

	if subscription.Events != nil {
		close(subscription.Events)
	}
	if subscription.UniqueEvents != nil {
		close(subscription.UniqueEvents)
	}
}

func (subscription Subscription) Sub(filter *filter.EventFilter) {
	if filter != nil {
		subscription.filter = filter
	}

	for _, ws := range subscription.relays {
		ws.WriteJSON([]interface{}{
			"REQ",
			subscription.channel,
			subscription.filter,
		})
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
		subscription.UniqueEvents <- em.Event
	}
}

func (subscription Subscription) removeRelay(relay string) {
	if ws, ok := subscription.relays[relay]; ok {
		delete(subscription.relays, relay)
		ws.WriteJSON([]interface{}{
			"CLOSE",
			subscription.channel,
		})
	}
}

func (subscription Subscription) addRelay(relay string, ws *websocket.Conn) {
	subscription.relays[relay] = ws
	ws.WriteJSON([]interface{}{
		"REQ",
		subscription.channel,
		subscription.filter,
	})
}
