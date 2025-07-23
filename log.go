package nostr

import (
	"log"
	"io"
)

var (
	// call SetOutput on InfoLogger to enable info logging
	InfoLogger = log.New(io.Discard, "[go-nostr][info] ", log.LstdFlags)

	// call SetOutput on DebugLogger to enable debug logging
	DebugLogger = log.New(io.Discard, "[go-nostr][debug] ", log.LstdFlags)
)
