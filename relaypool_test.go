package nostr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestRelayPoolSubUnique(t *testing.T) {
	// prepare test notes to send to a client subs
	priv, pub := makeKeyPair(t)
	notesMap := make(map[string]Event)
	notesFilter := Filter{}
	for i := 0; i < 10; i++ {
		note := Event{
			Kind:      1,
			Content:   fmt.Sprintf("hello %d", i),
			CreatedAt: time.Unix(1672068534+int64(i), 0),
			PubKey:    pub,
		}
		mustSignEvent(t, priv, &note)
		notesMap[note.ID] = note
		notesFilter.IDs = append(notesFilter.IDs, note.ID)
	}

	var mu sync.Mutex // guards subscribed and seenSubID to satisfy go test -race
	var (
		subscribed1, subscribed2 bool
		seenSubID1, seenSubID2   string
	)

	// fake relay server 1
	ws1 := newWebsocketServer(func(conn *websocket.Conn) {
		mu.Lock()
		subscribed1 = true
		mu.Unlock()
		// verify the client sent a good sub request
		var raw []json.RawMessage
		if err := websocket.JSON.Receive(conn, &raw); err != nil {
			t.Errorf("ws1: websocket.JSON.Receive: %v", err)
		}
		subid, filters := parseSubscriptionMessage(t, raw)
		seenSubID1 = subid
		if len(filters) != 1 || !FilterEqual(filters[0], notesFilter) {
			t.Errorf("ws1: client sent filters:\n%+v\nwant:\n%+v", filters, Filters{notesFilter})
		}
		// send back all the notes
		for id, note := range notesMap {
			if err := websocket.JSON.Send(conn, []any{"EVENT", subid, note}); err != nil {
				t.Errorf("ws1: %s: websocket.JSON.Send: %v", id, err)
			}
		}
	})
	defer ws1.Close()

	// fake relay server 2
	ws2 := newWebsocketServer(func(conn *websocket.Conn) {
		mu.Lock()
		subscribed2 = true
		mu.Unlock()
		// verify the client sent a good sub request
		var raw []json.RawMessage
		if err := websocket.JSON.Receive(conn, &raw); err != nil {
			t.Errorf("ws2: websocket.JSON.Receive: %v", err)
		}
		subid, filters := parseSubscriptionMessage(t, raw)
		seenSubID2 = subid
		if len(filters) != 1 || !FilterEqual(filters[0], notesFilter) {
			t.Errorf("ws2: client sent filters:\n%+v\nwant:\n%+v", filters, Filters{notesFilter})
		}
		// send back all the notes
		for id, note := range notesMap {
			if err := websocket.JSON.Send(conn, []any{"EVENT", subid, note}); err != nil {
				t.Errorf("ws2: %s: websocket.JSON.Send: %v", id, err)
			}
		}
	})
	defer ws2.Close()

	// connect a client, sub and verify it receives all events without duplicates
	pool := mustRelayPoolConnect(ws1.URL, ws2.URL)
	subid, ch, _ := pool.Sub(Filters{notesFilter})
	uniq := Unique(ch)

	seen := make(map[string]bool)
loop:
	for {
		select {
		case event := <-uniq:
			wantNote, ok := notesMap[event.ID]
			if !ok {
				t.Errorf("received unknown event: %+v", event)
				continue
			}
			if seen[event.ID] {
				t.Errorf("client already seen event %s", event.ID)
				continue
			}

			if !bytes.Equal(event.Serialize(), wantNote.Serialize()) {
				t.Errorf("received event:\n%+v\nwant:\n%+v", event, wantNote)
			}
			seen[event.ID] = true
			if len(seen) == len(notesMap) {
				break loop
			}
		case <-time.After(2 * time.Second):
			t.Errorf("took too long to receive from sub; seen %d out of %d events", len(seen), len(notesMap))
			break loop
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if !subscribed1 || !subscribed2 {
		t.Errorf("subscribed1=%v subscribed2=%v; want both true", subscribed1, subscribed2)
	}
	if seenSubID1 != subid || seenSubID2 != subid {
		t.Errorf("relay saw seenSubID1=%q seenSubID2=%q; want %q", seenSubID1, seenSubID2, subid)
	}
}

func mustRelayPoolConnect(url ...string) *RelayPool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool := NewRelayPool()
	readwrite := SimplePolicy{Read: true, Write: true}
	for _, u := range url {
		if err := pool.AddContext(ctx, u, readwrite); err != nil {
			panic(err.Error())
		}
	}
	return pool
}
