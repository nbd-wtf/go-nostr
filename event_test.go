package nostr

import (
	"encoding/json"
	"testing"
)

func TestEventParsingAndVerifying(t *testing.T) {
	raw := `{"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"kind":1,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}`

	var ev Event
	err := json.Unmarshal([]byte(raw), &ev)
	if err != nil {
		t.Errorf("failed to parse event json: %w", err)
	}

	if ev.GetID() != ev.ID {
		t.Errorf("error serializing event id: %s != %s", ev.GetID(), ev.ID)
	}

	if ok, _ := ev.CheckSignature(); !ok {
		t.Error("signature verification failed when it should have succeeded")
	}

	asjson, err := json.Marshal(ev)
	if err != nil {
		t.Errorf("failed to re marshal event as json: %w", err)
	}

	if string(asjson) != raw {
		t.Log(string(asjson))
		t.Error("json serialization broken")
	}
}
