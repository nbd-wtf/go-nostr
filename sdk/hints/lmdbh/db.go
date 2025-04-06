package lmdbh

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"slices"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/sdk/hints"
)

var _ hints.HintsDB = (*LMDBHints)(nil)

type LMDBHints struct {
	env *lmdb.Env
	dbi lmdb.DBI
}

func NewLMDBHints(path string) (*LMDBHints, error) {
	// create directory if it doesn't exist
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}

	// initialize environment
	env, err := lmdb.NewEnv()
	if err != nil {
		return nil, err
	}

	// set max DBs and map size
	env.SetMaxDBs(1)
	env.SetMapSize(1 << 30) // 1GB

	// open the environment
	if err := env.Open(path, lmdb.NoTLS|lmdb.WriteMap, 0644); err != nil {
		return nil, err
	}

	lh := &LMDBHints{env: env}

	// open the database
	if err := env.Update(func(txn *lmdb.Txn) error {
		dbi, err := txn.OpenDBI("hints", lmdb.Create)
		if err != nil {
			return err
		}
		lh.dbi = dbi
		return nil
	}); err != nil {
		env.Close()
		return nil, err
	}

	return lh, nil
}

func (lh *LMDBHints) Close() {
	lh.env.Close()
}

func (lh *LMDBHints) Save(pubkey string, relay string, hintkey hints.HintKey, ts nostr.Timestamp) {
	if now := nostr.Now(); ts > now {
		ts = now
	}

	err := lh.env.Update(func(txn *lmdb.Txn) error {
		k := encodeKey(pubkey, relay)
		var tss timestamps
		v, err := txn.Get(lh.dbi, k)
		if err == nil {
			// there is a value, so we may update it or not
			tss = parseValue(v)
		} else if !lmdb.IsNotFound(err) {
			return err
		}

		if tss[hintkey] < ts {
			tss[hintkey] = ts
			return txn.Put(lh.dbi, k, encodeValue(tss), 0)
		}

		return nil
	})
	if err != nil {
		nostr.InfoLogger.Printf("[sdk/hints/lmdb] unexpected error on save: %s\n", err)
	}
}

func (lh *LMDBHints) TopN(pubkey string, n int) []string {
	type relayScore struct {
		relay string
		score int64
	}

	scores := make([]relayScore, 0, n)
	err := lh.env.View(func(txn *lmdb.Txn) error {
		txn.RawRead = true

		cursor, err := txn.OpenCursor(lh.dbi)
		if err != nil {
			return err
		}
		defer cursor.Close()

		prefix, _ := hex.DecodeString(pubkey)
		k, v, err := cursor.Get(prefix, nil, lmdb.SetRange)
		for ; err == nil; k, v, err = cursor.Get(nil, nil, lmdb.Next) {
			// check if we're still in the prefix range
			if len(k) < 32 || !bytes.Equal(k[:32], prefix) {
				break
			}

			relay := string(k[32:])
			tss := parseValue(v)
			scores = append(scores, relayScore{relay, tss.sum()})
		}
		if err != nil && !lmdb.IsNotFound(err) {
			return err
		}
		return nil
	})
	if err != nil {
		nostr.InfoLogger.Printf("[sdk/hints/lmdb] unexpected error on topn: %s\n", err)
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

func (lh *LMDBHints) PrintScores() {
	fmt.Println("= print scores")

	err := lh.env.View(func(txn *lmdb.Txn) error {
		txn.RawRead = true

		cursor, err := txn.OpenCursor(lh.dbi)
		if err != nil {
			return err
		}
		defer cursor.Close()

		var lastPubkey string
		i := 0

		for k, v, err := cursor.Get(nil, nil, lmdb.First); err == nil; k, v, err = cursor.Get(nil, nil, lmdb.Next) {
			pubkey, relay := parseKey(k)

			if pubkey != lastPubkey {
				fmt.Println("== relay scores for", pubkey)
				lastPubkey = pubkey
				i = 0
			} else {
				i++
			}

			tss := parseValue(v)
			fmt.Printf("  %3d :: %30s ::> %12d\n", i, relay, tss.sum())
		}
		if !lmdb.IsNotFound(err) {
			return err
		}
		return nil
	})
	if err != nil {
		nostr.InfoLogger.Printf("[sdk/hints/lmdb] unexpected error on print: %s\n", err)
	}
}

func (lh *LMDBHints) GetDetailedScores(pubkey string, n int) []hints.RelayScores {
	type relayScore struct {
		relay string
		tss   timestamps
		score int64
	}

	scores := make([]relayScore, 0, n)
	err := lh.env.View(func(txn *lmdb.Txn) error {
		txn.RawRead = true

		cursor, err := txn.OpenCursor(lh.dbi)
		if err != nil {
			return err
		}
		defer cursor.Close()

		prefix, _ := hex.DecodeString(pubkey)
		k, v, err := cursor.Get(prefix, nil, lmdb.SetRange)
		for ; err == nil; k, v, err = cursor.Get(nil, nil, lmdb.Next) {
			// check if we're still in the prefix range
			if len(k) < 32 || !bytes.Equal(k[:32], prefix) {
				break
			}

			relay := string(k[32:])
			tss := parseValue(v)
			scores = append(scores, relayScore{relay, tss, tss.sum()})
		}
		if err != nil && !lmdb.IsNotFound(err) {
			return err
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

type timestamps [4]nostr.Timestamp

func (tss timestamps) sum() int64 {
	now := nostr.Now() + 24*60*60
	var sum int64
	for i, ts := range tss {
		if ts == 0 {
			continue
		}
		value := float64(hints.HintKey(i).BasePoints()) * 10000000000 / math.Pow(float64(max(now-ts, 1)), 1.3)
		sum += int64(value)
	}
	return sum
}
