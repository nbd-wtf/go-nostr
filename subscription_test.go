package nostr

import (
	"context"
	"testing"
	"time"
)

// test if we can connect to wss://relay.damus.io and fetch a couple of random events
func TestSubscribe(t *testing.T) {
	rl := mustRelayConnect("wss://relay.damus.io")
	defer rl.Close()

	sub, err := rl.Subscribe(context.Background(), Filters{{Kinds: []int{1}, Limit: 2}})
	if err != nil {
		t.Errorf("subscription failed: %v", err)
		return
	}

	timeout := time.After(5 * time.Second)
	events := 0

	for {
		select {
		case event := <-sub.Events:
			if event == nil {
				t.Errorf("event is nil: %v", event)
			}
			events++
		case <-sub.EndOfStoredEvents:
			goto end
		case <-rl.Context().Done():
			t.Errorf("connection closed: %v", rl.Context().Err())
			goto end
		case <-timeout:
			t.Errorf("timeout")
			goto end
		}
	}

end:
	if events != 2 {
		t.Errorf("expected 2 events, got %d", events)
	}
}
