package nostr

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEOSEMadness(t *testing.T) {
	rl := mustRelayConnect(t, RELAY)
	defer rl.Close()

	sub, err := rl.Subscribe(context.Background(), Filters{
		{Kinds: []int{KindTextNote}, Limit: 2},
	})
	assert.NoError(t, err)

	timeout := time.After(3 * time.Second)
	n := 0
	e := 0

	for {
		select {
		case event := <-sub.Events:
			assert.NotNil(t, event)
			n++
		case <-sub.EndOfStoredEvents:
			e++
			assert.Condition(t, func() (success bool) {
				return !(e > 1)
			}, "eose infinite loop")
			continue
		case <-rl.Context().Done():
			t.Fatalf("connection closed: %v", rl.Context().Err())
		case <-timeout:
			goto end
		}
	}

end:
	assert.Equal(t, 1, e)
	assert.Condition(t, func() (success bool) {
		return n >= 2
	})
}
