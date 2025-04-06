package badgerh

import (
	"encoding/hex"
	"fmt"
	"math"
	"slices"

	"github.com/dgraph-io/badger/v4"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/sdk/hints"
)

var _ hints.HintsDB = (*BadgerHints)(nil)

type BadgerHints struct {
	db *badger.DB
}

func NewBadgerHints(path string) (*BadgerHints, error) {
	opts := badger.DefaultOptions(path)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &BadgerHints{db: db}, nil
}

func (bh *BadgerHints) Close() {
	bh.db.Close()
}

func (bh *BadgerHints) Save(pubkey string, relay string, hintkey hints.HintKey, ts nostr.Timestamp) {
	if now := nostr.Now(); ts > now {
		ts = now
	}

	err := bh.db.Update(func(txn *badger.Txn) error {
		k := encodeKey(pubkey, relay)
		var tss timestamps
		item, err := txn.Get(k)
		if err == nil {
			err = item.Value(func(val []byte) error {
				// there is a value, so we may update it or not
				tss = parseValue(val)
				return nil
			})
			if err != nil {
				return err
			}
		} else if err != badger.ErrKeyNotFound {
			return err
		}

		if tss[hintkey] < ts {
			tss[hintkey] = ts
			return txn.Set(k, encodeValue(tss))
		}

		return nil
	})
	if err != nil {
		nostr.InfoLogger.Printf("[sdk/hints/badger] unexpected error on save: %s\n", err)
	}
}

func (bh *BadgerHints) TopN(pubkey string, n int) []string {
	type relayScore struct {
		relay string
		score int64
	}

	scores := make([]relayScore, 0, n)
	err := bh.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix, _ = hex.DecodeString(pubkey)
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(opts.Prefix); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			relay := string(k[32:])

			err := item.Value(func(val []byte) error {
				tss := parseValue(val)
				scores = append(scores, relayScore{relay, tss.sum()})
				return nil
			})
			if err != nil {
				continue
			}
		}
		return nil
	})
	if err != nil {
		nostr.InfoLogger.Printf("[sdk/hints/badger] unexpected error on topn: %s\n", err)
		return nil
	}

	slices.SortFunc(scores, func(a, b relayScore) int {
		return int(b.score - a.score)
	})

	result := make([]string, 0, n)
	for i, rs := range scores {
		if i >= n {
			break
		}
		result = append(result, rs.relay)
	}
	return result
}

func (bh *BadgerHints) GetDetailedScores(pubkey string, n int) []hints.RelayScores {
	type relayScore struct {
		relay string
		tss   timestamps
		score int64
	}

	scores := make([]relayScore, 0, n)
	err := bh.db.View(func(txn *badger.Txn) error {
		prefix, _ := hex.DecodeString(pubkey)
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			relay := string(k[32:])

			var tss timestamps
			err := item.Value(func(v []byte) error {
				tss = parseValue(v)
				return nil
			})
			if err != nil {
				return err
			}

			scores = append(scores, relayScore{relay, tss, tss.sum()})
		}
		return nil
	})
	if err != nil {
		return nil
	}

	slices.SortFunc(scores, func(a, b relayScore) int {
		return int(b.score - a.score)
	})

	result := make([]hints.RelayScores, 0, n)
	for i, rs := range scores {
		if i >= n {
			break
		}
		result = append(result, hints.RelayScores{
			Relay:  rs.relay,
			Scores: rs.tss,
			Sum:    rs.score,
		})
	}
	return result
}

func (bh *BadgerHints) PrintScores() {
	fmt.Println("= print scores")

	err := bh.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		var lastPubkey string
		i := 0

		for it.Seek(nil); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			pubkey, relay := parseKey(k)

			if pubkey != lastPubkey {
				fmt.Println("== relay scores for", pubkey)
				lastPubkey = pubkey
				i = 0
			} else {
				i++
			}

			err := item.Value(func(val []byte) error {
				tss := parseValue(val)
				fmt.Printf("  %3d :: %30s ::> %12d\n", i, relay, tss.sum())
				return nil
			})
			if err != nil {
				continue
			}
		}
		return nil
	})
	if err != nil {
		nostr.InfoLogger.Printf("[sdk/hints/badger] unexpected error on print: %s\n", err)
	}
}

type timestamps [4]nostr.Timestamp

func (tss timestamps) sum() int64 {
	now := nostr.Now() + 24*60*60
	var sum int64
	for i, ts := range tss {
		if ts == 0 {
			continue
		}
		value := float64(hints.HintKey(i).BasePoints()) * 10000000000 / math.Pow(float64(max(now-ts, 1)), 1.3)
		// fmt.Println("   ", i, "value:", value)
		sum += int64(value)
	}
	return sum
}
