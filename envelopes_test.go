package nostr

import (
	"encoding/json"
	"testing"
)

func TestEventEnvelopeEncodingAndDecoding(t *testing.T) {
	eventEnvelopes := []string{
		`["EVENT","_",{"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"kind":1,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}]`,
		`["EVENT",{"id":"9e662bdd7d8abc40b5b15ee1ff5e9320efc87e9274d8d440c58e6eed2dddfbe2","pubkey":"373ebe3d45ec91977296a178d9f19f326c70631d2a1b0bbba5c5ecc2eb53b9e7","created_at":1644844224,"kind":3,"tags":[["p","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"],["p","75fc5ac2487363293bd27fb0d14fb966477d0f1dbc6361d37806a6a740eda91e"],["p","46d0dfd3a724a302ca9175163bdf788f3606b3fd1bb12d5fe055d1e418cb60ea"]],"content":"{\"wss://nostr-pub.wellorder.net\":{\"read\":true,\"write\":true},\"wss://nostr.bitcoiner.social\":{\"read\":false,\"write\":true},\"wss://expensive-relay.fiatjaf.com\":{\"read\":true,\"write\":true},\"wss://relayer.fiatjaf.com\":{\"read\":true,\"write\":true},\"wss://relay.bitid.nz\":{\"read\":true,\"write\":true},\"wss://nostr.rocks\":{\"read\":true,\"write\":true}}","sig":"811355d3484d375df47581cb5d66bed05002c2978894098304f20b595e571b7e01b2efd906c5650080ffe49cf1c62b36715698e9d88b9e8be43029a2f3fa66be"}]`,
	}

	for _, raw := range eventEnvelopes {
		var env EventEnvelope
		err := json.Unmarshal([]byte(raw), &env)
		if err != nil {
			t.Errorf("failed to parse event envelope json: %v", err)
		}

		if env.GetID() != env.ID {
			t.Errorf("error serializing event id: %s != %s", env.GetID(), env.ID)
		}

		if ok, _ := env.CheckSignature(); !ok {
			t.Error("signature verification failed when it should have succeeded")
		}

		asjson, err := json.Marshal(env)
		if err != nil {
			t.Errorf("failed to re marshal event as json: %v", err)
		}

		if string(asjson) != raw {
			t.Log(string(asjson))
			t.Error("json serialization broken")
		}
	}
}

func TestNoticeEnvelopeEncodingAndDecoding(t *testing.T) {
	src := `["NOTICE","kjasbdlasvdluiasvd\"kjasbdksab\\d"]`
	var env NoticeEnvelope
	json.Unmarshal([]byte(src), &env)
	if env != "kjasbdlasvdluiasvd\"kjasbdksab\\d" {
		t.Error("failed to decode NOTICE")
	}

	res, _ := json.Marshal(env)
	if string(res) != src {
		t.Errorf("failed to encode NOTICE: expected '%s', got '%s'", src, string(res))
	}
}

func TestEoseEnvelopeEncodingAndDecoding(t *testing.T) {
	src := `["EOSE","kjasbdlasvdluiasvd\"kjasbdksab\\d"]`
	var env EOSEEnvelope
	json.Unmarshal([]byte(src), &env)
	if env != "kjasbdlasvdluiasvd\"kjasbdksab\\d" {
		t.Error("failed to decode EOSE")
	}

	res, _ := json.Marshal(env)
	if string(res) != src {
		t.Errorf("failed to encode EOSE: expected '%s', got '%s'", src, string(res))
	}
}

func TestOKEnvelopeEncodingAndDecoding(t *testing.T) {
	okEnvelopes := []string{
		`["OK","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefaaaaa",false,"error: could not connect to the database"]`,
		`["OK","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefaaaaa",true]`,
	}

	for _, raw := range okEnvelopes {
		var env OKEnvelope
		err := json.Unmarshal([]byte(raw), &env)
		if err != nil {
			t.Errorf("failed to parse ok envelope json: %v", err)
		}

		asjson, err := json.Marshal(env)
		if err != nil {
			t.Errorf("failed to re marshal ok as json: %v", err)
		}

		if string(asjson) != raw {
			t.Log(string(asjson))
			t.Error("json serialization broken")
		}
	}
}

func TestAuthEnvelopeEncodingAndDecoding(t *testing.T) {
	authEnvelopes := []string{
		`["AUTH","kjsabdlasb aslkd kasndkad \"as.kdnbskadb"]`,
		`["AUTH",{"id":"ae1fc7154296569d87ca4663f6bdf448c217d1590d28c85d158557b8b43b4d69","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","created_at":1683660344,"kind":1,"tags":[],"content":"hello world","sig":"94e10947814b1ebe38af42300ecd90c7642763896c4f69506ae97bfdf54eec3c0c21df96b7d95daa74ff3d414b1d758ee95fc258125deebc31df0c6ba9396a51"}]`,
	}

	for _, raw := range authEnvelopes {
		var env AuthEnvelope
		err := json.Unmarshal([]byte(raw), &env)
		if err != nil {
			t.Errorf("failed to parse auth envelope json: %v", err)
		}

		asjson, err := json.Marshal(env)
		if err != nil {
			t.Errorf("failed to re marshal auth as json: %v", err)
		}

		if string(asjson) != raw {
			t.Log(string(asjson))
			t.Error("json serialization broken")
		}
	}
}
