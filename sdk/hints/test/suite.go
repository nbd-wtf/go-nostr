package test

import (
	"testing"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/sdk/hints"
	"github.com/stretchr/testify/require"
)

func runTestWith(t *testing.T, hdb hints.HintsDB) {
	const key1 = "0000000000000000000000000000000000000000000000000000000000000001"
	const key2 = "0000000000000000000000000000000000000000000000000000000000000002"
	const key3 = "0000000000000000000000000000000000000000000000000000000000000003"
	const key4 = "0000000000000000000000000000000000000000000000000000000000000004"
	const relayA = "wss://aaa.com"
	const relayB = "wss://bbb.net"
	const relayC = "wss://ccc.org"

	hour := nostr.Timestamp((time.Hour).Seconds())
	day := hour * 24

	// key1: finding out
	// add some random parameters things and see what we get
	hdb.Save(key1, relayA, hints.LastInHint, nostr.Now()-60*hour)
	hdb.Save(key1, relayB, hints.LastInRelayList, nostr.Now()-day*10)
	hdb.Save(key1, relayB, hints.LastInHint, nostr.Now()-day*30)
	hdb.Save(key1, relayA, hints.LastInHint, nostr.Now()-hour*6)
	hdb.PrintScores()

	require.Equal(t, []string{relayB, relayA}, hdb.TopN(key1, 3))

	hdb.Save(key1, relayA, hints.LastFetchAttempt, nostr.Now()-5*hour)
	hdb.Save(key1, relayC, hints.LastInHint, nostr.Now()-5*hour)
	hdb.PrintScores()

	require.Equal(t, []string{relayB, relayC, relayA}, hdb.TopN(key1, 3))

	hdb.Save(key1, relayA, hints.LastInHint, nostr.Now()-1*hour)
	hdb.Save(key1, relayC, hints.LastFetchAttempt, nostr.Now()-5*hour)
	hdb.PrintScores()

	require.Equal(t, []string{relayB, relayA, relayC}, hdb.TopN(key1, 3))

	hdb.Save(key1, relayA, hints.MostRecentEventFetched, nostr.Now()-day*60)
	hdb.PrintScores()

	require.Equal(t, []string{relayB, relayA, relayC}, hdb.TopN(key1, 3))

	// now let's try a different thing for key2
	// key2 has a relay list with A and B
	hdb.Save(key2, relayA, hints.LastInRelayList, nostr.Now()-day*25)
	hdb.Save(key2, relayB, hints.LastInRelayList, nostr.Now()-day*25)

	// but it's old, recently we only see hints for relay C
	hdb.Save(key2, relayC, hints.LastInHint, nostr.Now()-4*hour)

	// at this point we just barely see C coming first
	hdb.PrintScores()
	require.Equal(t, []string{relayC, relayA, relayB}, hdb.TopN(key2, 3))

	// yet a different thing for key3
	// it doesn't have relay lists published because it's banned everywhere
	// all it has are references to its posts from others
	hdb.Save(key3, relayA, hints.LastInHint, nostr.Now()-day*2)
	hdb.Save(key3, relayB, hints.LastInHint, nostr.Now()-day)
	hdb.Save(key3, relayB, hints.LastInHint, nostr.Now()-day)
	hdb.PrintScores()
	require.Equal(t, []string{relayB, relayA}, hdb.TopN(key3, 3))

	// we try to fetch events for key3 and we get a very recent one for relay A, an older for relay B
	hdb.Save(key3, relayA, hints.LastFetchAttempt, nostr.Now()-5*hour)
	hdb.Save(key3, relayA, hints.MostRecentEventFetched, nostr.Now()-day)
	hdb.Save(key3, relayB, hints.LastFetchAttempt, nostr.Now()-5*hour)
	hdb.Save(key3, relayB, hints.MostRecentEventFetched, nostr.Now()-day*30)
	hdb.PrintScores()
	require.Equal(t, []string{relayA, relayB}, hdb.TopN(key3, 3))

	// for key4 we'll try the alex jones case
	// key4 used to publish normally to a bunch of big relays until it got banned
	// then it started publishing only to its personal relay
	// how long until clients realize that?
	banDate := nostr.Now() - day*10
	hdb.Save(key4, relayA, hints.LastInRelayList, banDate)
	hdb.Save(key4, relayA, hints.LastFetchAttempt, banDate)
	hdb.Save(key4, relayA, hints.MostRecentEventFetched, banDate)
	hdb.Save(key4, relayA, hints.LastInHint, banDate+12*day)
	hdb.Save(key4, relayB, hints.LastInRelayList, banDate)
	hdb.Save(key4, relayB, hints.LastFetchAttempt, banDate)
	hdb.Save(key4, relayB, hints.MostRecentEventFetched, banDate)
	hdb.Save(key4, relayB, hints.LastInHint, banDate+2*day)
	hdb.PrintScores()
	require.Equal(t, []string{relayA, relayB}, hdb.TopN(key4, 3))

	// information about the new relay starts to spread through relay hints in tags only
	hdb.Save(key4, relayC, hints.LastInHint, nostr.Now()-3*day)

	// as long as we see one tag hint the new relay will already be in our map
	hdb.PrintScores()
	require.Equal(t, []string{relayA, relayB, relayC}, hdb.TopN(key4, 3))

	// client tries to fetch stuff from the old relays, but gets nothing new
	hdb.Save(key4, relayA, hints.LastFetchAttempt, nostr.Now()-5*hour)
	hdb.Save(key4, relayB, hints.LastFetchAttempt, nostr.Now()-5*hour)

	// which is enough for us to transition to the new relay as the toppermost of the uppermost
	hdb.PrintScores()
	require.Equal(t, []string{relayC, relayA, relayB}, hdb.TopN(key4, 3))

	// what if the big relays are attempting to game this algorithm by allowing some of our
	// events from time to time while still shadowbanning us?
	hdb.Save(key4, relayA, hints.MostRecentEventFetched, nostr.Now()-5*hour)
	hdb.Save(key4, relayB, hints.MostRecentEventFetched, nostr.Now()-5*hour)
	hdb.PrintScores()
	require.Equal(t, []string{relayA, relayB, relayC}, hdb.TopN(key4, 3))

	// we'll need overwhelming force from the third relay
	// (actually just a relay list with just its name in it will be enough)
	hdb.Save(key4, relayC, hints.LastFetchAttempt, nostr.Now()-5*hour)
	hdb.Save(key4, relayC, hints.MostRecentEventFetched, nostr.Now()-6*hour)
	hdb.Save(key4, relayC, hints.LastInRelayList, nostr.Now()-6*hour)
	hdb.PrintScores()
	require.Equal(t, []string{relayC, relayA, relayB}, hdb.TopN(key4, 3))

	//
	//
	// things remain the same for key1, key2 and key3
	require.Equal(t, []string{relayC, relayA}, hdb.TopN(key2, 2))
	require.Equal(t, []string{relayB, relayA, relayC}, hdb.TopN(key1, 3))
	require.Equal(t, []string{relayA, relayB}, hdb.TopN(key3, 3))
}
