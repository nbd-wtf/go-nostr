package nostr

import (
	"context"
	"testing"
	"time"
)

const RELAY = "wss://relay.damus.io"

// test if we can fetch a couple of random events
func TestSubscribe(t *testing.T) {
	rl := mustRelayConnect(RELAY)
	defer rl.Close()

	sub, err := rl.Subscribe(context.Background(), Filters{{Kinds: []int{1}, Limit: 2}})
	if err != nil {
		t.Errorf("subscription failed: %v", err)
		return
	}

	timeout := time.After(5 * time.Second)
	n := 0

	for {
		select {
		case event := <-sub.Events:
			if event == nil {
				t.Errorf("event is nil: %v", event)
			}
			n++
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
	if n != 2 {
		t.Errorf("expected 2 events, got %d", n)
	}
}

// test if we can do multiple nested subscriptions
func TestNestedSubscriptions(t *testing.T) {
	rl := mustRelayConnect(RELAY)
	defer rl.Close()

	// fetch any note
	sub, err := rl.Subscribe(context.Background(), Filters{{Kinds: []int{1}, Limit: 1}})
	if err != nil {
		t.Errorf("subscription 1 failed: %v", err)
		return
	}

	timeout := time.After(5 * time.Second)

	for {
		select {
		case event := <-sub.Events:
			// now fetch author of this event
			sub, err := rl.Subscribe(context.Background(), Filters{{Kinds: []int{0}, Authors: []string{event.PubKey}, Limit: 1}})
			if err != nil {
				t.Errorf("subscription 2 failed: %v", err)
				return
			}

			for {
				select {
				case <-sub.Events:
					// now mentions of this person
					sub, err := rl.Subscribe(context.Background(), Filters{{Kinds: []int{1}, Tags: TagMap{"p": []string{event.PubKey}}, Limit: 1}})
					if err != nil {
						t.Errorf("subscription 3 failed: %v", err)
						return
					}

					for {
						select {
						case <-sub.Events:
							// if we get here safely we won
							return
						case <-timeout:
							t.Errorf("timeout 3")
						}
					}
				case <-timeout:
					t.Errorf("timeout 2")
				}
			}
		case <-rl.Context().Done():
			t.Errorf("connection closed: %v", rl.Context().Err())
		case <-timeout:
			t.Errorf("timeout 1")
		}
	}
}
