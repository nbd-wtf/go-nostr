package negentropy

import (
	"fmt"
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestSmallNumber(t *testing.T) {
	var err error
	var q []byte
	var n1 *Negentropy
	var n2 *Negentropy

	events := make([]*nostr.Event, 20)
	for i := range events {
		evt := nostr.Event{Content: fmt.Sprintf("event %d", i+1)}
		evt.CreatedAt = nostr.Timestamp(i)
		evt.ID = evt.GetID()
		events[i] = &evt
	}

	{
		n1, _ = NewNegentropy(NewVector(32), 1<<16, 32)
		for i := 2; i < 15; i++ {
			n1.Insert(events[i])
		}

		q, err = n1.Initiate()
		if err != nil {
			t.Fatal(err)
			return
		}

		fmt.Println("[n1]:", q)
	}

	{
		n2, _ = NewNegentropy(NewVector(32), 1<<16, 32)
		for i := 0; i < 7; i++ {
			n2.Insert(events[i])
		}
		for i := 10; i < 20; i++ {
			n2.Insert(events[i])
		}

		q, _, _, err = n2.Reconcile(q)
		if err != nil {
			t.Fatal(err)
			return
		}
		fmt.Println("[n2]:", q)
	}

	{
		var have []string
		var need []string
		q, have, need, err = n1.Reconcile(q)
		if err != nil {
			t.Fatal(err)
			return
		}
		fmt.Println("[n1]:", q)
		fmt.Println("")
		fmt.Println("have", have)
		fmt.Println("need", need)

		expectedNeed := make([]string, 0, 2+5)
		for i := 0; i < 2; i++ {
			expectedNeed = append(expectedNeed, events[i].ID)
		}
		for i := 15; i < 20; i++ {
			expectedNeed = append(expectedNeed, events[i].ID)
		}

		expectedHave := make([]string, 0, 3)
		for i := 7; i < 10; i++ {
			expectedHave = append(expectedHave, events[i].ID)
		}

		require.ElementsMatch(t, expectedNeed, need, "wrong need")
		require.ElementsMatch(t, expectedHave, have, "wrong have")
	}
}
