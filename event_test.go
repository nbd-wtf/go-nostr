package nostr

import (
	"encoding/json"
	"testing"
)

func TestEventParsingAndVerifying(t *testing.T) {
	rawEvents := []string{
		`{"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"kind":1,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}`,
		`{"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"kind":1,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524","extrakey":55}`,
		`{"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"kind":1,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524","extrakey":"aaa"}`,
		`{"id":"9e662bdd7d8abc40b5b15ee1ff5e9320efc87e9274d8d440c58e6eed2dddfbe2","pubkey":"373ebe3d45ec91977296a178d9f19f326c70631d2a1b0bbba5c5ecc2eb53b9e7","created_at":1644844224,"kind":3,"tags":[["p","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"],["p","75fc5ac2487363293bd27fb0d14fb966477d0f1dbc6361d37806a6a740eda91e"],["p","46d0dfd3a724a302ca9175163bdf788f3606b3fd1bb12d5fe055d1e418cb60ea"]],"content":"{\"wss://nostr-pub.wellorder.net\":{\"read\":true,\"write\":true},\"wss://nostr.bitcoiner.social\":{\"read\":false,\"write\":true},\"wss://expensive-relay.fiatjaf.com\":{\"read\":true,\"write\":true},\"wss://relayer.fiatjaf.com\":{\"read\":true,\"write\":true},\"wss://relay.bitid.nz\":{\"read\":true,\"write\":true},\"wss://nostr.rocks\":{\"read\":true,\"write\":true}}","sig":"811355d3484d375df47581cb5d66bed05002c2978894098304f20b595e571b7e01b2efd906c5650080ffe49cf1c62b36715698e9d88b9e8be43029a2f3fa66be"}`,
	}

	for _, raw := range rawEvents {
		var ev Event
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
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
		if err != nil {
			t.Log(evt)
			t.Error("failed to serialize this event")
		}

		var re Event
		if err := json.Unmarshal(b, &re); err != nil {
			t.Log(string(b))
			t.Error("failed to re parse event just serialized")
		}

		if evt.ID != re.ID || evt.PubKey != re.PubKey || evt.Content != re.Content ||
			evt.CreatedAt != re.CreatedAt || evt.Sig != re.Sig ||
			len(evt.Tags) != len(re.Tags) {
			t.Error("reparsed event differs from original")
		}

		for i := range evt.Tags {
			if len(evt.Tags[i]) != len(re.Tags[i]) {
				t.Errorf("reparsed tags %d length differ from original", i)
				continue
			}

			for j := range evt.Tags[i] {
				if evt.Tags[i][j] != re.Tags[i][j] {
					t.Errorf("reparsed tag content %d %d length differ from original", i, j)
				}
			}
		}
	}
}

func TestEventSerializationWithExtraFields(t *testing.T) {
	evt := Event{
		ID:        "92570b321da503eac8014b23447301eb3d0bbdfbace0d11a4e4072e72bb7205d",
		PubKey:    "e9142f724955c5854de36324dab0434f97b15ec6b33464d56ebe491e3f559d1b",
		Kind:      KindReaction,
		CreatedAt: Timestamp(1671028682),
		Content:   "there is an extra field here",
		Sig:       "ed08d2dd5b0f7b6a3cdc74643d4adee3158ddede9cc848e8cd97630c097001acc2d052d2d3ec2b7ac4708b2314b797106d1b3c107322e61b5e5cc2116e099b79",
	}
	evt.SetExtra("glub", true)
	evt.SetExtra("plik", nil)
	evt.SetExtra("elet", 77)
	evt.SetExtra("malf", "hello")

	b, err := json.Marshal(evt)
	if err != nil {
		t.Log(evt)
		t.Error("failed to serialize this event")
	}

	var re Event
	if err := json.Unmarshal(b, &re); err != nil {
		t.Log(string(b))
		t.Error("failed to re parse event just serialized")
	}

	if evt.ID != re.ID || evt.PubKey != re.PubKey || evt.Content != re.Content ||
		evt.CreatedAt != re.CreatedAt || evt.Sig != re.Sig ||
		len(evt.Tags) != len(re.Tags) {
		t.Error("reparsed event differs from original")
	}

	if evt.GetExtra("malf").(string) != evt.GetExtraString("malf") || evt.GetExtraString("malf") != "hello" {
		t.Errorf("failed to parse extra string")
	}

	if float64(evt.GetExtra("elet").(int)) != evt.GetExtraNumber("elet") || evt.GetExtraNumber("elet") != 77 {
		t.Logf("number: %v == %v", evt.GetExtra("elet"), evt.GetExtraNumber("elet"))
		t.Errorf("failed to parse extra number")
	}

	if evt.GetExtra("glub").(bool) != evt.GetExtraBoolean("glub") || evt.GetExtraBoolean("glub") != true {
		t.Errorf("failed to parse extra boolean")
	}

	if evt.GetExtra("plik") != nil {
		t.Errorf("failed to parse extra null")
	}
}

func mustSignEvent(t *testing.T, privkey string, event *Event) {
	t.Helper()
	if err := event.Sign(privkey); err != nil {
		t.Fatalf("event.Sign: %v", err)
	}
}
