package nostr

import (
	"encoding/json"
	"testing"
)

func TestEventParsingAndVerifying(t *testing.T) {
	rawEvents := []string{
		`{"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"kind":1,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}`,
		`{"id":"9e662bdd7d8abc40b5b15ee1ff5e9320efc87e9274d8d440c58e6eed2dddfbe2","pubkey":"373ebe3d45ec91977296a178d9f19f326c70631d2a1b0bbba5c5ecc2eb53b9e7","created_at":1644844224,"kind":3,"tags":[["p","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"],["p","75fc5ac2487363293bd27fb0d14fb966477d0f1dbc6361d37806a6a740eda91e"],["p","46d0dfd3a724a302ca9175163bdf788f3606b3fd1bb12d5fe055d1e418cb60ea"]],"content":"{\"wss://nostr-pub.wellorder.net\":{\"read\":true,\"write\":true},\"wss://nostr.bitcoiner.social\":{\"read\":false,\"write\":true},\"wss://expensive-relay.fiatjaf.com\":{\"read\":true,\"write\":true},\"wss://relayer.fiatjaf.com\":{\"read\":true,\"write\":true},\"wss://relay.bitid.nz\":{\"read\":true,\"write\":true},\"wss://nostr.rocks\":{\"read\":true,\"write\":true}}","sig":"811355d3484d375df47581cb5d66bed05002c2978894098304f20b595e571b7e01b2efd906c5650080ffe49cf1c62b36715698e9d88b9e8be43029a2f3fa66be"}`,
	}

	for _, raw := range rawEvents {
		var ev Event
		err := json.Unmarshal([]byte(raw), &ev)
		if err != nil {
			t.Errorf("failed to parse event json: %v", err)
		}

		if ev.GetID() != ev.ID {
			t.Errorf("error serializing event id: %s != %s", ev.GetID(), ev.ID)
		}

		if ok, _ := ev.CheckSignature(); !ok {
			t.Error("signature verification failed when it should have succeeded")
		}

		asjson, err := json.Marshal(ev)
		if err != nil {
			t.Errorf("failed to re marshal event as json: %v", err)
		}

		if string(asjson) != raw {
			t.Log(string(asjson))
			t.Error("json serialization broken")
		}
	}
}
