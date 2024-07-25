package negentropy

import (
	"encoding/hex"
	"fmt"
	"strings"
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

func TestMuchBiggerNumbersAndConfusion(t *testing.T) {
	runTestWith(t,
		20000,
		[][]int{{20, 150}, {1700, 3400}, {7000, 8100}, {13800, 13816}, {13817, 14950}, {19800, 20000}}, // n1
		[][]int{{0, 2000}, {3000, 3600}, {10000, 12200}, {13799, 13801}, {14800, 19900}},               // n2
		[][]int{{0, 20}, {150, 1700}, {3400, 3600}, {10000, 12200}, {13799, 13800}, {14950, 19800}},    // n1 need
		[][]int{{2000, 3000}, {7000, 8100}, {13801, 13816}, {13817, 14800}, {19900, 20000}},            // n1 have
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
		evt.Content = fmt.Sprintf("event %d", i)
		evt.Kind = 1
		evt.CreatedAt = nostr.Timestamp(i)
		evt.ID = fmt.Sprintf("%064d", i)
		events[i] = &evt
	}

	{
		n1, _ = NewNegentropy(NewVector(), 1<<16)
		for _, r := range n1Ranges {
			for i := r[0]; i < r[1]; i++ {
				n1.Insert(events[i])
			}
		}

		q = n1.Initiate()

		fmt.Println("[n1]:", len(q)) //, hexedBytes(q))
		fmt.Println("")
	}

	{
		n2, _ = NewNegentropy(NewVector(), 1<<16)
		for _, r := range n2Ranges {
			for i := r[0]; i < r[1]; i++ {
				n2.Insert(events[i])
			}
		}

		q, _, _, err = n2.Reconcile(1, q)
		if err != nil {
			t.Fatal(err)
			return
		}
		fmt.Println("[n2]:", len(q)) // , hexedBytes(q))
		fmt.Println("")
	}

	invert := map[*Negentropy]*Negentropy{
		n1: n2,
		n2: n1,
	}
	i := 1
	for n := n1; q != nil; n = invert[n] {
		i++
		var have []string
		var need []string

		q, have, need, err = n.Reconcile(i, q)
		if err != nil {
			t.Fatal(err)
			return
		}
		fmt.Println("[n-]:", len(q)) //, hexedBytes(q))
		fmt.Println("")

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

func hexedBytes(o []byte) string {
	s := strings.Builder{}
	s.Grow(2 + 1 + len(o)*5)
	s.WriteString("[ ")
	for _, b := range o {
		x := hex.EncodeToString([]byte{b})
		s.WriteString("0x")
		s.WriteString(x)
		s.WriteString(" ")
	}
	s.WriteString("]")
	return s.String()
}
