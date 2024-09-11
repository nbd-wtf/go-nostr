package memory

import (
	"fmt"
	"math"
	"slices"
	"sync"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/sdk/hints"
)

var _ hints.HintsDB = (*HintDB)(nil)

type HintDB struct {
	RelayBySerial         []string
	OrderedRelaysByPubKey map[string]RelaysForPubKey

	sync.Mutex
}

func NewHintDB() *HintDB {
	return &HintDB{
		RelayBySerial:         make([]string, 0, 100),
		OrderedRelaysByPubKey: make(map[string]RelaysForPubKey, 100),
	}
}

func (db *HintDB) Save(pubkey string, relay string, key hints.HintKey, ts nostr.Timestamp) {
	now := nostr.Now()
	// this is used for calculating what counts as a usable hint
	threshold := (now - 60*60*24*180)
	if threshold < 0 {
		threshold = 0
	}

	relayIndex := slices.Index(db.RelayBySerial, relay)
	if relayIndex == -1 {
		relayIndex = len(db.RelayBySerial)
		db.RelayBySerial = append(db.RelayBySerial, relay)
	}

	db.Lock()
	defer db.Unlock()
	// fmt.Println(" ", relay, "index", relayIndex, "--", "adding", hints.HintKey(key).String(), ts)

	rfpk, _ := db.OrderedRelaysByPubKey[pubkey]

	entries := rfpk.Entries

	entryIndex := slices.IndexFunc(entries, func(re RelayEntry) bool { return re.Relay == relayIndex })
	if entryIndex == -1 {
		// we don't have an entry for this relay, so add one
		entryIndex = len(entries)

		entry := RelayEntry{
			Relay: relayIndex,
		}
		entry.Timestamps[key] = ts

		entries = append(entries, entry)
	} else {
		// just update this entry
		if entries[entryIndex].Timestamps[key] < ts {
			entries[entryIndex].Timestamps[key] = ts
		} else {
			// no need to update anything
			return
		}
	}

	rfpk.Entries = entries

	db.OrderedRelaysByPubKey[pubkey] = rfpk
}

func (db *HintDB) TopN(pubkey string, n int) []string {
	db.Lock()
	defer db.Unlock()

	urls := make([]string, 0, n)
	if rfpk, ok := db.OrderedRelaysByPubKey[pubkey]; ok {
		// sort everything from scratch
		slices.SortFunc(rfpk.Entries, func(a, b RelayEntry) int {
			return int(b.Sum() - a.Sum())
		})

		for i, re := range rfpk.Entries {
			urls = append(urls, db.RelayBySerial[re.Relay])
			if i+1 == n {
				break
			}
		}
	}
	return urls
}

func (db *HintDB) PrintScores() {
	db.Lock()
	defer db.Unlock()

	fmt.Println("= print scores")
	for pubkey, rfpk := range db.OrderedRelaysByPubKey {
		fmt.Println("== relay scores for", pubkey)
		for i, re := range rfpk.Entries {
			fmt.Printf("  %3d :: %30s (%3d) ::> %12d\n", i, db.RelayBySerial[re.Relay], re.Relay, re.Sum())
		}
	}
}

type RelaysForPubKey struct {
	Entries []RelayEntry
}

type RelayEntry struct {
	Relay      int
	Timestamps [8]nostr.Timestamp
}

func (re RelayEntry) Sum() int64 {
	now := nostr.Now() + 24*60*60
	var sum int64
	for i, ts := range re.Timestamps {
		if ts == 0 {
			continue
		}

		hk := hints.HintKey(i)
		divisor := int64(now - ts)
		if divisor == 0 {
			divisor = 1
		} else {
			divisor = int64(math.Pow(float64(divisor), 1.3))
		}

		multiplier := hk.BasePoints()
		value := multiplier * 10000000000 / divisor
		// fmt.Println("   ", i, "value:", value)
		sum += value
	}
	return sum
}
