package negentropy

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

const FingerprintSize = 16

type Mode uint8

const (
	SkipMode        Mode = 0
	FingerprintMode Mode = 1
	IdListMode      Mode = 2
)

func (v Mode) String() string {
	switch v {
	case SkipMode:
		return "SKIP"
	case FingerprintMode:
		return "FINGERPRINT"
	case IdListMode:
		return "IDLIST"
	default:
		return "<UNKNOWN-ERROR>"
	}
}

type Item struct {
	Timestamp nostr.Timestamp
	ID        string
}

func ItemCompare(a, b Item) int {
	if a.Timestamp == b.Timestamp {
		return strings.Compare(a.ID, b.ID)
	}
	return cmp.Compare(a.Timestamp, b.Timestamp)
}

func (i Item) String() string { return fmt.Sprintf("Item<%d:%s>", i.Timestamp, i.ID) }

type Bound struct{ Item }

func (b Bound) String() string {
	if b.Timestamp == InfiniteBound.Timestamp {
		return "Bound<infinite>"
	}
	return fmt.Sprintf("Bound<%d:%s>", b.Timestamp, b.ID)
}
