package nip45

import (
	"iter"
	"strconv"

	"github.com/nbd-wtf/go-nostr"
)

func HyperLogLogEventPubkeyOffsetsAndReferencesForEvent(evt *nostr.Event) iter.Seq2[string, int] {
	return func(yield func(string, int) bool) {
		switch evt.Kind {
		case 3:
			//
			// follower counts
			for _, tag := range evt.Tags {
				if len(tag) >= 2 && tag[0] == "p" && nostr.IsValid32ByteHex(tag[1]) {
					// 32th nibble of each "p" tag
					p, _ := strconv.ParseInt(tag[1][32:33], 16, 64)
					if !yield(tag[1], int(p+8)) {
						return
					}
				}
			}
		case 7:
			//
			// reaction counts:
			// (only the last "e" tag counts)
			lastE := evt.Tags.FindLast("e")
			if lastE != nil {
				v := lastE[1]
				if nostr.IsValid32ByteHex(v) {
					// 32th nibble of "e" tag
					p, _ := strconv.ParseInt(v[32:33], 16, 64)
					if !yield(v, int(p+8)) {
						return
					}
				}
			}
		case 1111:
			//
			// comment counts:
			e := evt.Tags.Find("E")
			if e != nil {
				v := e[1]
				if nostr.IsValid32ByteHex(v) {
					// 32th nibble of "e" tag
					p, _ := strconv.ParseInt(v[32:33], 16, 64)
					if !yield(v, int(p+8)) {
						return
					}
				}
			}
		}
	}
}
