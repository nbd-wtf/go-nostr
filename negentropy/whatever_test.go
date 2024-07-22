package negentropy

import (
	"fmt"
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestSuperSmall(t *testing.T) {
	runTestWith(t,
		4,
		[][]int{{0, 3}}, [][]int{{2, 4}},
		[][]int{{3, 4}}, [][]int{{0, 2}},
	)
}

func TestNoNeedToSync(t *testing.T) {
	runTestWith(t,
		50,
		[][]int{{0, 50}}, [][]int{{0, 50}},
		[][]int{}, [][]int{},
	)
}

func TestSmallNumbers(t *testing.T) {
	runTestWith(t,
		20,
		[][]int{{2, 15}}, [][]int{{0, 7}, {10, 20}},
		[][]int{{0, 2}, {15, 20}}, [][]int{{7, 10}},
	)
}

func TestBigNumbers(t *testing.T) {
	runTestWith(t,
		200,
		[][]int{{20, 150}}, [][]int{{0, 70}, {100, 200}},
		[][]int{{0, 20}, {150, 200}}, [][]int{{70, 100}},
	)
}

func runTestWith(t *testing.T,
	totalEvents int,
	n1Ranges [][]int, n2Ranges [][]int,
	expectedN1NeedRanges [][]int, expectedN1HaveRanges [][]int,
) {
	var err error
	var q []byte
	var n1 *Negentropy
	var n2 *Negentropy

	events := make([]*nostr.Event, totalEvents)
	for i := range events {
		evt := nostr.Event{}
		evt.Content = fmt.Sprintf("event %d", i+1)
		evt.Kind = 1
		evt.CreatedAt = nostr.Timestamp(i)
		evt.ID = evt.GetID()
		events[i] = &evt
		fmt.Println("evt", i, evt.ID)
	}

	{
		n1, _ = NewNegentropy(NewVector(32), 1<<16, 32)
		for _, r := range n1Ranges {
			for i := r[0]; i < r[1]; i++ {
				n1.Insert(events[i])
			}
		}

		q, err = n1.Initiate()
		if err != nil {
			t.Fatal(err)
			return
		}

		fmt.Println("[n1]:", len(q), q)
	}

	{
		n2, _ = NewNegentropy(NewVector(32), 1<<16, 32)
		for _, r := range n2Ranges {
			for i := r[0]; i < r[1]; i++ {
				n2.Insert(events[i])
			}
		}

		q, _, _, err = n2.Reconcile(q)
		if err != nil {
			t.Fatal(err)
			return
		}
		fmt.Println("[n2]:", len(q), q)
	}

	invert := map[*Negentropy]*Negentropy{
		n1: n2,
		n2: n1,
	}

	for n := n1; q != nil; n = invert[n] {
		var have []string
		var need []string

		q, have, need, err = n.Reconcile(q)
		if err != nil {
			t.Fatal(err)
			return
		}
		fmt.Println("[n-]:", len(q), q)

		if q == nil {
			fmt.Println("")
			// fmt.Println("<need>", need)
			// fmt.Println("<have>", have)

			expectedNeed := make([]string, 0, 100)
			for _, r := range expectedN1NeedRanges {
				for i := r[0]; i < r[1]; i++ {
					expectedNeed = append(expectedNeed, events[i].ID)
				}
			}

			expectedHave := make([]string, 0, 100)
			for _, r := range expectedN1HaveRanges {
				for i := r[0]; i < r[1]; i++ {
					expectedHave = append(expectedHave, events[i].ID)
				}
			}

			// fmt.Println("<e-need>", expectedNeed)
			// fmt.Println("<e-have>", expectedHave)

			require.ElementsMatch(t, expectedNeed, need, "wrong need")
			require.ElementsMatch(t, expectedHave, have, "wrong have")
		}
	}
}
