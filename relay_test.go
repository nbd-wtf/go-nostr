package nostr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestPublish(t *testing.T) {
	// test note to be sent over websocket
	priv, pub := makeKeyPair(t)
	textNote := Event{
		Kind:      KindTextNote,
		Content:   "hello",
		CreatedAt: Timestamp(1672068534), // random fixed timestamp
		Tags:      Tags{[]string{"foo", "bar"}},
		PubKey:    pub,
	}
	if err := textNote.Sign(priv); err != nil {
		t.Fatalf("textNote.Sign: %v", err)
	}

	// fake relay server
	var mu sync.Mutex // guards published to satisfy go test -race
	var published bool
	ws := newWebsocketServer(func(conn *websocket.Conn) {
		mu.Lock()
		published = true
		mu.Unlock()
		// verify the client sent exactly the textNote
		var raw []json.RawMessage
		if err := websocket.JSON.Receive(conn, &raw); err != nil {
			t.Errorf("websocket.JSON.Receive: %v", err)
		}
		event := parseEventMessage(t, raw)
		if !bytes.Equal(event.Serialize(), textNote.Serialize()) {
			t.Errorf("received event:\n%+v\nwant:\n%+v", event, textNote)
		}
		// send back an ok nip-20 command result
		res := []any{"OK", textNote.ID, true, ""}
		if err := websocket.JSON.Send(conn, res); err != nil {
			t.Errorf("websocket.JSON.Send: %v", err)
		}
	})
	defer ws.Close()

	// connect a client and send the text note
	rl := mustRelayConnect(ws.URL)
	err := rl.Publish(context.Background(), textNote)
	if err != nil {
		t.Errorf("publish should have succeeded")
	}
	if !published {
		t.Errorf("fake relay server saw no event")
	}
}

func TestPublishBlocked(t *testing.T) {
	// test note to be sent over websocket
	textNote := Event{Kind: KindTextNote, Content: "hello"}
	textNote.ID = textNote.GetID()

	// fake relay server
	ws := newWebsocketServer(func(conn *websocket.Conn) {
		// discard received message; not interested
		var raw []json.RawMessage
		if err := websocket.JSON.Receive(conn, &raw); err != nil {
			t.Errorf("websocket.JSON.Receive: %v", err)
		}
		// send back a not ok nip-20 command result
		res := []any{"OK", textNote.ID, false, "blocked"}
		websocket.JSON.Send(conn, res)
	})
	defer ws.Close()

	// connect a client and send a text note
	rl := mustRelayConnect(ws.URL)
	err := rl.Publish(context.Background(), textNote)
	if err == nil {
		t.Errorf("should have failed to publish")
	}
}

func TestPublishWriteFailed(t *testing.T) {
	// test note to be sent over websocket
	textNote := Event{Kind: KindTextNote, Content: "hello"}
	textNote.ID = textNote.GetID()

	// fake relay server
	ws := newWebsocketServer(func(conn *websocket.Conn) {
		// reject receive - force send error
		conn.Close()
	})
	defer ws.Close()

	// connect a client and send a text note
	rl := mustRelayConnect(ws.URL)
	// Force brief period of time so that publish always fails on closed socket.
	time.Sleep(1 * time.Millisecond)
	err := rl.Publish(context.Background(), textNote)
	if err == nil {
		t.Errorf("should have failed to publish")
	}
}

func TestConnectContext(t *testing.T) {
	// fake relay server
	var mu sync.Mutex // guards connected to satisfy go test -race
	var connected bool
	ws := newWebsocketServer(func(conn *websocket.Conn) {
		mu.Lock()
		connected = true
		mu.Unlock()
		io.ReadAll(conn) // discard all input
	})
	defer ws.Close()

	// relay client
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	r, err := RelayConnect(ctx, ws.URL)
	if err != nil {
		t.Fatalf("RelayConnectContext: %v", err)
	}
	defer r.Close()

	mu.Lock()
	defer mu.Unlock()
	if !connected {
		t.Error("fake relay server saw no client connect")
	}
}

func TestConnectContextCanceled(t *testing.T) {
	// fake relay server
	ws := newWebsocketServer(discardingHandler)
	defer ws.Close()

	// relay client
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // make ctx expired
	_, err := RelayConnect(ctx, ws.URL)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("RelayConnectContext returned %v error; want context.Canceled", err)
	}
}

func TestConnectWithOrigin(t *testing.T) {
	// fake relay server
	// default handler requires origin golang.org/x/net/websocket
	ws := httptest.NewServer(websocket.Handler(discardingHandler))
	defer ws.Close()

	// relay client
	r := NewRelay(context.Background(), NormalizeURL(ws.URL))
	r.RequestHeader = http.Header{"origin": {"https://example.com"}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.Connect(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEventIDIntegrity(t *testing.T) {

	priv, pub := makeKeyPair(t)

	makeEvent := func(timestamp int) Event {
		textNote := Event{
			Kind:      KindTextNote,
			Content:   "hello",
			CreatedAt: Timestamp(timestamp),
			Tags:      Tags{[]string{"foo", "bar"}},
			PubKey:    pub,
		}
		if err := textNote.Sign(priv); err != nil {
			t.Fatalf("textNote.Sign: %v", err)
		}

		return textNote
	}

	tests := []struct {
		name        string
		event       Event
		shouldFail  bool
		modifyEvent func(event *Event)
	}{
		{
			name:        "Original event ID",
			event:       makeEvent(1672068534),
			shouldFail:  false,
			modifyEvent: nil, // No modification needed for the original event
		},
		{
			name:        "Altered event ID",
			event:       makeEvent(1672068535),
			shouldFail:  true,
			modifyEvent: func(event *Event) { event.ID = "fake_id" }, // Alter the ID to simulate failure
		},
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {

			if tc.modifyEvent != nil {
				tc.modifyEvent(&tc.event)
			}

			ctx, cancel := context.WithCancel(context.Background())
			wg := sync.WaitGroup{}
			wg.Add(1)

			// fake relay server
			ws := newWebsocketServer(func(conn *websocket.Conn) {

				var raw []json.RawMessage
				// handling publich event
				if err := websocket.JSON.Receive(conn, &raw); err != nil {
					t.Errorf("websocket.JSON.Receive: %v", err)
				}

				// send back an ok nip-20 command result
				res := []any{"OK", tc.event.ID, true, ""}
				if err := websocket.JSON.Send(conn, res); err != nil {
					t.Errorf("websocket.JSON.Send: %v", err)
				}

				// handling subscription event
				if err := websocket.JSON.Receive(conn, &raw); err != nil {
					t.Errorf("websocket.JSON.Receive: %v", err)
				}

				subID, _, err := parseSubscriptionMessage(t, raw)
				if err == nil {
					res := EventEnvelope{
						SubscriptionID: &subID,
						Event:          tc.event,
					}
					// senc back the EVENT packet
					if err := websocket.JSON.Send(conn, res); err != nil {
						t.Errorf("websocket.JSON.Send: %v", err)
					}
				}

				wg.Wait()
			})

			rl := mustRelayConnect(ws.URL)

			go func() {

				defer cancel()

				sub, err := rl.Subscribe(context.Background(), Filters{{Kinds: []int{KindTextNote}, Limit: 1}})

				if err != nil {
					t.Logf("subscription.err = %v", err)
					return
				}

				select {
				case event := <-sub.Events:
					if tc.shouldFail && event != nil {
						t.Errorf("Unexpected event received: ID=%v", event.ID)
					}
					if !tc.shouldFail && event == nil {
						t.Error("Expected event not received")
					}
				case <-time.After(time.Second * 2):
					if !tc.shouldFail {
						t.Error("Event timeout")
					}
				}

				wg.Done()

			}()

			// connect a client and send the text note
			err := rl.Publish(context.Background(), tc.event)
			if err != nil {
				t.Errorf("publish should have succeeded. err=%v", err)
			}

			<-ctx.Done()
			ws.Close()
		})

	}

}

func discardingHandler(conn *websocket.Conn) {
	io.ReadAll(conn) // discard all input
}

func newWebsocketServer(handler func(*websocket.Conn)) *httptest.Server {
	return httptest.NewServer(&websocket.Server{
		Handshake: anyOriginHandshake,
		Handler:   handler,
	})
}

// anyOriginHandshake is an alternative to default in golang.org/x/net/websocket
// which checks for origin. nostr client sends no origin and it makes no difference
// for the tests here anyway.
var anyOriginHandshake = func(conf *websocket.Config, r *http.Request) error {
	return nil
}

func makeKeyPair(t *testing.T) (priv, pub string) {
	t.Helper()
	privkey := GeneratePrivateKey()
	pubkey, err := GetPublicKey(privkey)
	if err != nil {
		t.Fatalf("GetPublicKey(%q): %v", privkey, err)
	}
	return privkey, pubkey
}

func mustRelayConnect(url string) *Relay {
	rl, err := RelayConnect(context.Background(), url)
	if err != nil {
		panic(err.Error())
	}
	return rl
}

func parseEventMessage(t *testing.T, raw []json.RawMessage) Event {
	t.Helper()
	if len(raw) < 2 {
		t.Fatalf("len(raw) = %d; want at least 2", len(raw))
	}
	var typ string
	json.Unmarshal(raw[0], &typ)
	if typ != "EVENT" {
		t.Errorf("typ = %q; want EVENT", typ)
	}
	var event Event
	if err := json.Unmarshal(raw[1], &event); err != nil {
		t.Errorf("json.Unmarshal(`%s`): %v", string(raw[1]), err)
	}
	return event
}

func parseSubscriptionMessage(t *testing.T, raw []json.RawMessage) (string, []Filter, error) {
	t.Helper()
	if len(raw) < 3 {
		return "", nil, fmt.Errorf("len(raw) = %d; want at least 3", len(raw))
	}
	var typ string
	json.Unmarshal(raw[0], &typ)
	if typ != "REQ" {
		return "", nil, fmt.Errorf("typ = %q; want REQ", typ)
	}
	var id string
	if err := json.Unmarshal(raw[1], &id); err != nil {
		return "", nil, fmt.Errorf("json.Unmarshal sub id: %v", err)
	}
	var ff []Filter
	for i, b := range raw[2:] {
		var f Filter
		if err := json.Unmarshal(b, &f); err != nil {
			return "", nil, fmt.Errorf("json.Unmarshal filter %d: %v", i, err)
		}
		ff = append(ff, f)
	}
	return id, ff, nil
}
