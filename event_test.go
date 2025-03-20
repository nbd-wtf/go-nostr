package nostr

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventParsingAndVerifying(t *testing.T) {
	rawEvents := []string{
		`{"kind":1,"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}`,
		`{"kind":1,"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}`,
		`{"kind":1,"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}`,
		`{"kind":3,"id":"9e662bdd7d8abc40b5b15ee1ff5e9320efc87e9274d8d440c58e6eed2dddfbe2","pubkey":"373ebe3d45ec91977296a178d9f19f326c70631d2a1b0bbba5c5ecc2eb53b9e7","created_at":1644844224,"tags":[["p","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"],["p","75fc5ac2487363293bd27fb0d14fb966477d0f1dbc6361d37806a6a740eda91e"],["p","46d0dfd3a724a302ca9175163bdf788f3606b3fd1bb12d5fe055d1e418cb60ea"]],"content":"{\"wss://nostr-pub.wellorder.net\":{\"read\":true,\"write\":true},\"wss://nostr.bitcoiner.social\":{\"read\":false,\"write\":true},\"wss://expensive-relay.fiatjaf.com\":{\"read\":true,\"write\":true},\"wss://relayer.fiatjaf.com\":{\"read\":true,\"write\":true},\"wss://relay.bitid.nz\":{\"read\":true,\"write\":true},\"wss://nostr.rocks\":{\"read\":true,\"write\":true}}","sig":"811355d3484d375df47581cb5d66bed05002c2978894098304f20b595e571b7e01b2efd906c5650080ffe49cf1c62b36715698e9d88b9e8be43029a2f3fa66be"}`,
	}

	for _, raw := range rawEvents {
		var ev Event
		err := json.Unmarshal([]byte(raw), &ev)
		assert.NoError(t, err)

		assert.Equal(t, ev.ID, ev.GetID())

		ok, _ := ev.CheckSignature()
		assert.True(t, ok, "signature verification failed when it should have succeeded")

		asJSON, err := json.Marshal(ev)
		assert.NoError(t, err)
		assert.Equal(t, raw, string(asJSON))
	}
}

func TestEventSerialization(t *testing.T) {
	events := []Event{
		{
			ID:        "92570b321da503eac8014b23447301eb3d0bbdfbace0d11a4e4072e72bb7205d",
			PubKey:    "e9142f724955c5854de36324dab0434f97b15ec6b33464d56ebe491e3f559d1b",
			Kind:      KindEncryptedDirectMessage,
			CreatedAt: Timestamp(1671028682),
			Tags:      Tags{Tag{"p", "f8340b2bde651576b75af61aa26c80e13c65029f00f7f64004eece679bf7059f"}},
			Content:   "you say yes, I say no",
			Sig:       "ed08d2dd5b0f7b6a3cdc74643d4adee3158ddede9cc848e8cd97630c097001acc2d052d2d3ec2b7ac4708b2314b797106d1b3c107322e61b5e5cc2116e099b79",
		},
	}

	for _, evt := range events {
		b, err := json.Marshal(evt)
		assert.NoError(t, err)

		var re Event
		err = json.Unmarshal(b, &re)
		assert.NoError(t, err)

		assert.Condition(t, func() (success bool) {
			if evt.ID != re.ID || evt.PubKey != re.PubKey || evt.Content != re.Content ||
				evt.CreatedAt != re.CreatedAt || evt.Sig != re.Sig ||
				len(evt.Tags) != len(re.Tags) {
				return false
			}
			return true
		}, "re-parsed event differs from original")

		for i := range evt.Tags {
			assert.Equal(t, len(evt.Tags[i]), len(re.Tags[i]), "re-parsed tags %d length differ from original", i)

			for j := range evt.Tags[i] {
				assert.Equal(t, re.Tags[i][j], evt.Tags[i][j], "re-parsed tag content %d %d length differ from original", i, j)
			}
		}
	}
}

func mustSignEvent(t *testing.T, privkey string, event *Event) {
	t.Helper()
	if err := event.Sign(privkey); err != nil {
		t.Fatalf("event.Sign: %v", err)
	}
}

func TestIDCheck(t *testing.T) {
	for i := 0; i < 1000; i++ {
		evt := Event{
			CreatedAt: Timestamp(rand.Int64N(9999999)),
			Content:   fmt.Sprintf("hello %d", i),
			Tags:      Tags{},
		}

		require.False(t, evt.CheckID())

		evt.Sign(GeneratePrivateKey())
		require.True(t, evt.CheckID())

		evt.Content += "!"
		require.False(t, evt.CheckID())
	}
}

func BenchmarkIDCheck(b *testing.B) {
	evt := Event{
		CreatedAt: Timestamp(rand.Int64N(9999999)),
		Content:   fmt.Sprintf("hello"),
		Tags:      Tags{},
	}
	evt.Sign(GeneratePrivateKey())

	b.Run("naïve", func(b *testing.B) {
		for b.Loop() {
			_ = evt.GetID() == evt.ID
		}
	})

	b.Run("big brain", func(b *testing.B) {
		for b.Loop() {
			_ = evt.CheckID()
		}
	})
}
