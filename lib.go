package nostr

import (
	"io/ioutil"
	"log"
)

var (
	// call SetOutput on InfoLogger to enable info logging
	InfoLogger = log.New(ioutil.Discard, "[go-nostr][info] ", log.LstdFlags)

	// call SetOutput on DebugLogger to enable debug logging
	DebugLogger = log.New(ioutil.Discard, "[go-nostr][debug] ", log.LstdFlags)
)
