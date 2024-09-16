package nip19

import (
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeNpub(t *testing.T) {
	npub, err := EncodePublicKey("3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d")
	assert.NoError(t, err)
	assert.Equal(t, "npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6", npub, "produced an unexpected npub string")
}

func TestEncodeNsec(t *testing.T) {
	nsec, err := EncodePrivateKey("3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d")
	assert.NoError(t, err)
	assert.Equal(t, "nsec180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsgyumg0", nsec, "produced an unexpected nsec string")
}

func TestDecodeNpub(t *testing.T) {
	prefix, pubkey, err := Decode("npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6")
	assert.NoError(t, err)
	assert.Equal(t, "npub", prefix, "returned invalid prefix")
	assert.Equal(t, "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d", pubkey.(string), "returned wrong pubkey")
}

func TestFailDecodeBadChecksumNpub(t *testing.T) {
	_, _, err := Decode("npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w4")
	assert.Error(t, err)
}

func TestDecodeNprofile(t *testing.T) {
	t.Run("first", func(t *testing.T) {
		prefix, data, err := Decode("nprofile1qqsrhuxx8l9ex335q7he0f09aej04zpazpl0ne2cgukyawd24mayt8gpp4mhxue69uhhytnc9e3k7mgpz4mhxue69uhkg6nzv9ejuumpv34kytnrdaksjlyr9p")
		assert.NoError(t, err)
		assert.Equal(t, "nprofile", prefix)

		pp, ok := data.(nostr.ProfilePointer)
		assert.True(t, ok, "value returned of wrong type")
		assert.Equal(t, "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d", pp.PublicKey)
		assert.Equal(t, 2, len(pp.Relays), "decoded wrong number of relays")

		assert.Equal(t, "wss://r.x.com", pp.Relays[0], "decoded relay URLs wrongly")
		assert.Equal(t, "wss://djbas.sadkb.com", pp.Relays[1], "decoded relay URLs wrongly")
	})

	t.Run("second", func(t *testing.T) {
		prefix, data, err := Decode("nprofile1qqsw3dy8cpumpanud9dwd3xz254y0uu2m739x0x9jf4a9sgzjshaedcpr4mhxue69uhkummnw3ez6ur4vgh8wetvd3hhyer9wghxuet5qyw8wumn8ghj7mn0wd68yttjv4kxz7fww4h8get5dpezumt9qyvhwumn8ghj7un9d3shjetj9enxjct5dfskvtnrdakstl69hg")
		assert.NoError(t, err)
		assert.Equal(t, "nprofile", prefix)

		pp, ok := data.(nostr.ProfilePointer)
		assert.True(t, ok, "value returned of wrong type")
		assert.Equal(t, "e8b487c079b0f67c695ae6c4c2552a47f38adfa2533cc5926bd2c102942fdcb7", pp.PublicKey)
		assert.Equal(t, 3, len(pp.Relays), "decoded wrong number of relays")

		assert.Equal(t, "wss://nostr-pub.wellorder.net", pp.Relays[0], "decoded relay URLs wrongly")
		assert.Equal(t, "wss://nostr-relay.untethr.me", pp.Relays[1], "decoded relay URLs wrongly")
	})
}

func TestEncodeNprofile(t *testing.T) {
	nprofile, err := EncodeProfile("3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d", []string{
		"wss://r.x.com",
		"wss://djbas.sadkb.com",
	})

	assert.NoError(t, err)
	assert.Equal(t,
		"nprofile1qqsrhuxx8l9ex335q7he0f09aej04zpazpl0ne2cgukyawd24mayt8gpp4mhxue69uhhytnc9e3k7mgpz4mhxue69uhkg6nzv9ejuumpv34kytnrdaksjlyr9p",
		nprofile, "produced an unexpected nprofile string: %s", nprofile)
}

func TestEncodeDecodeNaddr(t *testing.T) {
	naddr, err := EncodeEntity(
		"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d",
		nostr.KindArticle,
		"banana",
		[]string{
			"wss://relay.nostr.example.mydomain.example.com",
			"wss://nostr.banana.com",
		})

	assert.NoError(t, err)
	assert.Equal(t,
		"naddr1qqrxyctwv9hxzqfwwaehxw309aex2mrp0yhxummnw3ezuetcv9khqmr99ekhjer0d4skjm3wv4uxzmtsd3jjucm0d5q3vamnwvaz7tmwdaehgu3wvfskuctwvyhxxmmdqgsrhuxx8l9ex335q7he0f09aej04zpazpl0ne2cgukyawd24mayt8grqsqqqa28a3lkds",
		naddr, "produced an unexpected naddr string: %s", naddr)

	prefix, data, err := Decode(naddr)
	assert.NoError(t, err)
	assert.Equal(t, "naddr", prefix)

	ep := data.(nostr.EntityPointer)
	assert.Equal(t, "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d", ep.PublicKey)
	assert.Equal(t, ep.Kind, nostr.KindArticle)
	assert.Equal(t, "banana", ep.Identifier)

	assert.Equal(t, "wss://relay.nostr.example.mydomain.example.com", ep.Relays[0])
	assert.Equal(t, "wss://nostr.banana.com", ep.Relays[1])
}

func TestDecodeNaddrWithoutRelays(t *testing.T) {
	prefix, data, err := Decode("naddr1qq98yetxv4ex2mnrv4esygrl54h466tz4v0re4pyuavvxqptsejl0vxcmnhfl60z3rth2xkpjspsgqqqw4rsf34vl5")
	assert.NoError(t, err, "unexpected error during decoding of Naddr")
	assert.Equal(t, "naddr", prefix, "returned invalid prefix")

	ep, ok := data.(nostr.EntityPointer)
	assert.True(t, ok)
	assert.Equal(t, "7fa56f5d6962ab1e3cd424e758c3002b8665f7b0d8dcee9fe9e288d7751ac194", ep.PublicKey)
	assert.Equal(t, nostr.KindArticle, ep.Kind)
	assert.Equal(t, "references", ep.Identifier)
	assert.Empty(t, ep.Relays)
}

func TestEncodeDecodeNEvent(t *testing.T) {
	nevent, err := EncodeEvent(
		"45326f5d6962ab1e3cd424e758c3002b8665f7b0d8dcee9fe9e288d7751ac194",
		[]string{"wss://banana.com"},
		"7fa56f5d6962ab1e3cd424e758c3002b8665f7b0d8dcee9fe9e288d7751abb88",
	)

	assert.NoError(t, err)

	expectedNEvent := "nevent1qqsy2vn0t45k92c78n2zfe6ccvqzhpn977cd3h8wnl579zxhw5dvr9qpzpmhxue69uhkyctwv9hxztnrdaksygrl54h466tz4v0re4pyuavvxqptsejl0vxcmnhfl60z3rth2x4m3q04ndyp"
	assert.Equal(t, expectedNEvent, nevent)

	prefix, res, err := Decode(nevent)
	assert.NoError(t, err)

	assert.Equal(t, "nevent", prefix)

	ep, ok := res.(nostr.EventPointer)
	assert.True(t, ok)

	assert.Equal(t, "7fa56f5d6962ab1e3cd424e758c3002b8665f7b0d8dcee9fe9e288d7751abb88", ep.Author)
	assert.Equal(t, "45326f5d6962ab1e3cd424e758c3002b8665f7b0d8dcee9fe9e288d7751ac194", ep.ID)
	assert.Equal(t, 1, len(ep.Relays), "wrong number of relays")
	assert.Equal(t, "wss://banana.com", ep.Relays[0])
}

func TestFailDecodeBadlyFormattedPubkey(t *testing.T) {
	_, _, err := Decode("nevent1qqsgaj0la08u0vl2ecmlmrg4xl0vjcz647yx7jgvgzfr566ael4hmjgpp4mhxue69uhhjctzw5hx6egzgqurswpc8qurswpexq6rjvm9xp3nvcfkv56xzv35v9jnxve389snqephve3n2wf4vdsnxepcv56kxct9xyunjdf5v5cnzveexqcrsepnk6yu5r")
	require.Error(t, err, "should fail to decode this because the author is hex as bytes garbage")
}
