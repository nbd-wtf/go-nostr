package hints

import "github.com/nbd-wtf/go-nostr"

const END_OF_WORLD nostr.Timestamp = 2208999600 // 2040-01-01

type HintKey int

const (
	LastFetchAttempt HintKey = iota
	MostRecentEventFetched
	LastInRelayList
	LastInHint
)

var KeyBasePoints = [4]int64{
	-500, // attempting has negative power because it may fail
	700,  // when it succeeds that should cancel the negative effect of trying
	350,  // a relay list is a very strong indicator
	20,   // hints from various sources (tags, nprofile, nevent, nip05)
}

func (hk HintKey) BasePoints() int64 { return KeyBasePoints[hk] }

func (hk HintKey) String() string {
	switch hk {
	case LastFetchAttempt:
		return "last_fetch_attempt"
	case MostRecentEventFetched:
		return "most_recent_event_fetched"
	case LastInRelayList:
		return "last_in_relay_list"
	case LastInHint:
		return "last_in_hint"
	}
	return "<unexpected>"
}
