package relays

import (
	"context"
	"testing"
	"time"

	"github.com/nbd-wtf/go-nostr/core"
)

func TestEOSEMadness(t *testing.T) {
	rl := mustRelayConnect(RELAY)
	defer rl.Close()

	sub, err := rl.Subscribe(context.Background(), core.Filters{
		{Kinds: []int{core.KindTextNote}, Limit: 2},
	})
	if err != nil {
		t.Errorf("subscription failed: %v", err)
		return
	}

	timeout := time.After(3 * time.Second)
	n := 0
	e := 0

	for {
		select {
		case event := <-sub.Events:
			if event == nil {
				t.Fatalf("event is nil: %v", event)
			}
			n++
		case <-sub.EndOfStoredEvents:
			e++
			if e > 1 {
				t.Fatalf("eose infinite loop")
			}
			continue
		case <-rl.Context().Done():
			t.Fatalf("connection closed: %v", rl.Context().Err())
		case <-timeout:
			goto end
		}
	}

end:
	if e != 1 {
		t.Fatalf("didn't get an eose")
	}
	if n < 2 {
		t.Fatalf("didn't get events")
	}
}
