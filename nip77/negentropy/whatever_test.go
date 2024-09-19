package negentropy_test

import (
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip77/negentropy"
	"github.com/nbd-wtf/go-nostr/nip77/negentropy/storage/vector"
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
	var q string
	var n1 *negentropy.Negentropy
	var n2 *negentropy.Negentropy

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
		n1s := vector.New()
		n1 = negentropy.New(n1s, 1<<16)
		for _, r := range n1Ranges {
			for i := r[0]; i < r[1]; i++ {
				n1s.Insert(events[i].CreatedAt, events[i].ID)
			}
		}
		n1s.Seal()

		q = n1.Start()
	}

	{
		n2s := vector.New()
		n2 = negentropy.New(n2s, 1<<16)
		for _, r := range n2Ranges {
			for i := r[0]; i < r[1]; i++ {
				n2s.Insert(events[i].CreatedAt, events[i].ID)
			}
		}
		n2s.Seal()

		q, err = n2.Reconcile(q)
		if err != nil {
			t.Fatal(err)
			return
		}
	}

	invert := map[*negentropy.Negentropy]*negentropy.Negentropy{
		n1: n2,
		n2: n1,
	}
	i := 1

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		expectedHave := make([]string, 0, 100)
		for _, r := range expectedN1HaveRanges {
			for i := r[0]; i < r[1]; i++ {
				expectedHave = append(expectedHave, events[i].ID)
			}
		}
		haves := make([]string, 0, 100)
		for item := range n1.Haves {
			if slices.Contains(haves, item) {
				continue
			}
			haves = append(haves, item)
		}
		slices.Sort(haves)
		require.Equal(t, expectedHave, haves, "wrong have")
	}()

	go func() {
		defer wg.Done()
		expectedNeed := make([]string, 0, 100)
		for _, r := range expectedN1NeedRanges {
			for i := r[0]; i < r[1]; i++ {
				expectedNeed = append(expectedNeed, events[i].ID)
			}
		}
		havenots := make([]string, 0, 100)
		for item := range n1.HaveNots {
			if slices.Contains(havenots, item) {
				continue
			}
			havenots = append(havenots, item)
		}
		slices.Sort(havenots)
		require.Equal(t, expectedNeed, havenots, "wrong need")
	}()

	for n := n1; q != ""; n = invert[n] {
		i++

		q, err = n.Reconcile(q)
		if err != nil {
			t.Fatalf("reconciliation failed: %s", err)
		}

		if q == "" {
			wg.Wait()
			return
		}
	}
}
