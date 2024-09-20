package negentropy_test

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip77/negentropy"
	"github.com/nbd-wtf/go-nostr/nip77/negentropy/storage/vector"
	"github.com/stretchr/testify/require"
)

func FuzzWhatever(f *testing.F) {
	var sectors uint = 1
	var sectorSizeAvg uint = 10
	var pctChance uint = 5
	var frameSizeLimit uint = 0
	f.Add(sectors, sectorSizeAvg, pctChance, frameSizeLimit)
	f.Fuzz(func(t *testing.T, sectors uint, sectorSizeAvg uint, pctChance uint, frameSizeLimit uint) {
		rand := rand.New(rand.NewPCG(1, 1000))
		sectorSizeAvg += 1 // prevent divide by zero
		frameSizeLimit += 4096
		pctChance = pctChance % 100

		// prepare the two sides
		s1 := vector.New()
		l1 := make([]string, 0, 500)
		neg1 := negentropy.New(s1, int(frameSizeLimit))
		s2 := vector.New()
		l2 := make([]string, 0, 500)
		neg2 := negentropy.New(s2, int(frameSizeLimit))

		start := 0
		for s := 0; s < int(sectors); s++ {
			diff := rand.Uint() % sectorSizeAvg
			if rand.IntN(2) == 0 {
				diff = -diff
			}
			sectorSize := sectorSizeAvg + diff

			for i := 0; i < int(sectorSize); i++ {
				item := start + i

				rnd := sha256.Sum256(binary.BigEndian.AppendUint64(nil, uint64(item)))
				id := fmt.Sprintf("%x%056d", rnd[0:4], item)

				if rand.IntN(100) < int(pctChance) {
					s1.Insert(nostr.Timestamp(item), id)
					l1 = append(l1, id)
				}
				if rand.IntN(100) < int(pctChance) {
					id := fmt.Sprintf("%064d", item)
					s2.Insert(nostr.Timestamp(item), id)
					l2 = append(l2, id)
				}
			}

			start += int(sectorSize)
		}

		// fmt.Println(neg1.Name(), "initial", l1)
		// fmt.Println(neg2.Name(), "initial", l2)

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			for item := range neg1.Haves {
				// fmt.Println("have", item)
				l2 = append(l2, item)
			}
			wg.Done()
		}()
		go func() {
			for item := range neg1.HaveNots {
				// fmt.Println("havenot", item)
				l1 = append(l1, item)
			}
			wg.Done()
		}()

		msg := neg1.Start()
		next := neg2

		for {
			var err error
			// fmt.Println(next.Name(), "handling", msg)
			msg, err = next.Reconcile(msg)
			if err != nil {
				panic(err)
			}

			if msg == "" {
				break
			}

			if next == neg1 {
				next = neg2
			} else {
				next = neg1
			}
		}

		wg.Wait()
		slices.Sort(l1)
		l1 = slices.Compact(l1)
		slices.Sort(l2)
		l2 = slices.Compact(l2)
		require.ElementsMatch(t, l1, l2)
	})
}
