package sdk

import (
	"context"
	"strconv"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	cache_memory "github.com/nbd-wtf/go-nostr/sdk/cache/memory"
)

type EventRef struct{ nostr.Pointer }

func (e EventRef) Value() string { return e.Pointer.AsTagReference() }

func (sys *System) FetchBookmarkList(ctx context.Context, pubkey string) GenericList[EventRef] {
	if sys.BookmarkListCache == nil {
		sys.BookmarkListCache = cache_memory.New32[GenericList[EventRef]](1000)
	}

	ml, _ := fetchGenericList(sys, ctx, pubkey, 10003, kind_10003, parseEventRef, sys.BookmarkListCache)
	return ml
}

func (sys *System) FetchPinList(ctx context.Context, pubkey string) GenericList[EventRef] {
	if sys.PinListCache == nil {
		sys.PinListCache = cache_memory.New32[GenericList[EventRef]](1000)
	}

	ml, _ := fetchGenericList(sys, ctx, pubkey, 10001, kind_10001, parseEventRef, sys.PinListCache)
	return ml
}

func parseEventRef(tag nostr.Tag) (evr EventRef, ok bool) {
	if len(tag) < 2 {
		return evr, false
	}
	switch tag[0] {
	case "e":
		if !nostr.IsValid32ByteHex(tag[1]) {
			return evr, false
		}
		pointer := nostr.EventPointer{
			ID: tag[1],
		}
		if len(tag) >= 3 {
			pointer.Relays = []string{nostr.NormalizeURL(tag[2])}
			if len(tag) >= 4 {
				pointer.Author = tag[3]
			}
		}
		evr.Pointer = pointer
	case "a":
		spl := strings.SplitN(tag[1], ":", 3)
		if len(spl) != 3 || !nostr.IsValidPublicKey(spl[1]) {
			return evr, false
		}
		pointer := nostr.EntityPointer{
			PublicKey:  spl[1],
			Identifier: spl[2],
		}
		if kind, err := strconv.Atoi(spl[0]); err != nil {
			return evr, false
		} else {
			pointer.Kind = kind
		}
		if len(tag) >= 3 {
			pointer.Relays = []string{nostr.NormalizeURL(tag[2])}
		}
		evr.Pointer = pointer
	default:
		return evr, false
	}

	return evr, false
}
