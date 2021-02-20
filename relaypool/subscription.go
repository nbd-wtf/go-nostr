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
}

func (subscription Subscription) Unsub() {
	for _, ws := range subscription.relays {
		ws.WriteJSON([]interface{}{
			"CLOSE",
			subscription.channel,
		})
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

type EventMessage struct {
	Event event.Event
	Relay string
}
