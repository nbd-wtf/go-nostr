package test

import (
	"os"
	"testing"

	"github.com/nbd-wtf/go-nostr/sdk/hints/badgerh"
)

func TestBadgerHints(t *testing.T) {
	path := "/tmp/tmpsdkhintsbadger"
	os.RemoveAll(path)

	hdb, err := badgerh.NewBadgerHints(path)
	if err != nil {
		t.Fatal(err)
	}
	defer hdb.Close()

	runTestWith(t, hdb)
}
