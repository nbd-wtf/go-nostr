package hints

import "github.com/nbd-wtf/go-nostr"

type RelayScores struct {
	Relay  string
	Scores [4]nostr.Timestamp
	Sum    int64
}

type HintsDB interface {
	TopN(pubkey string, n int) []string
	Save(pubkey string, relay string, key HintKey, score nostr.Timestamp)
	PrintScores()
	GetDetailedScores(pubkey string, n int) []RelayScores
}
