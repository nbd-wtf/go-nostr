package nson

import (
	"encoding/json"
	"testing"

	"github.com/mailru/easyjson"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/test_common"
)

func TestBasicNsonParse(t *testing.T) {
	for _, jevt := range nsonTestEvents {
		evt := &nostr.Event{}
		if err := Unmarshal(jevt, evt); err != nil {
			t.Fatalf("error unmarshalling nson: %s", err)
		}
		checkParsedCorrectly(t, evt, jevt)
	}
}

func TestNsonPartialGet(t *testing.T) {
	for _, jevt := range nsonTestEvents {
		evt := &nostr.Event{}
		if err := Unmarshal(jevt, evt); err != nil {
			t.Fatalf("error unmarshalling nson: %s", err)
		}

		wrapper := New(jevt)

		if id := wrapper.GetID(); id != evt.ID {
			t.Fatalf("partial id wrong. got %v, expected %v", id, evt.ID)
		}
		if pubkey := wrapper.GetPubkey(); pubkey != evt.PubKey {
			t.Fatalf("partial pubkey wrong. got %v, expected %v", pubkey, evt.PubKey)
		}
		if sig := wrapper.GetSig(); sig != evt.Sig {
			t.Fatalf("partial sig wrong. got %v, expected %v", sig, evt.Sig)
		}
		if createdAt := wrapper.GetCreatedAt(); createdAt != evt.CreatedAt {
			t.Fatalf("partial created_at wrong. got %v, expected %v", createdAt, evt.CreatedAt)
		}
		if kind := wrapper.GetKind(); kind != evt.Kind {
			t.Fatalf("partial kind wrong. got %v, expected %v", kind, evt.Kind)
		}
		if content := wrapper.GetContent(); content != evt.Content {
			t.Fatalf("partial content wrong. got %v, expected %v", content, evt.Content)
		}
	}
}

func TestNsonEncode(t *testing.T) {
	for _, jevt := range test_common.NormalEvents {
		pevt := &nostr.Event{}
		if err := json.Unmarshal([]byte(jevt), pevt); err != nil {
			t.Fatalf("failed to decode normal json: %s", err)
		}
		nevt, err := Marshal(pevt)
		if err != nil {
			t.Fatalf("failed to encode nson: %s", err)
		}

		evt := &nostr.Event{}
		if err := Unmarshal(nevt, evt); err != nil {
			t.Fatalf("error unmarshalling nson: %s", err)
		}
		checkParsedCorrectly(t, pevt, jevt)
		checkParsedCorrectly(t, evt, jevt)
	}
}

func checkParsedCorrectly(t *testing.T, evt *nostr.Event, jevt string) (isBad bool) {
	var canonical nostr.Event
	err := json.Unmarshal([]byte(jevt), &canonical)
	if err != nil {
		t.Fatalf("error unmarshalling normal json: %s", err)
	}

	if evt.ID != canonical.ID {
		t.Fatalf("id is wrong: %s != %s", evt.ID, canonical.ID)
		isBad = true
	}
	if evt.PubKey != canonical.PubKey {
		t.Fatalf("pubkey is wrong: %s != %s", evt.PubKey, canonical.PubKey)
		isBad = true
	}
	if evt.Sig != canonical.Sig {
		t.Fatalf("sig is wrong: %s != %s", evt.Sig, canonical.Sig)
		isBad = true
	}
	if evt.Content != canonical.Content {
		t.Fatalf("content is wrong: %s != %s", evt.Content, canonical.Content)
		isBad = true
	}
	if evt.Kind != canonical.Kind {
		t.Fatalf("kind is wrong: %d != %d", evt.Kind, canonical.Kind)
		isBad = true
	}
	if evt.CreatedAt != canonical.CreatedAt {
		t.Fatalf("created_at is wrong: %v != %v", evt.CreatedAt, canonical.CreatedAt)
		isBad = true
	}
	if len(evt.Tags) != len(canonical.Tags) {
		t.Fatalf("tag number is wrong: %v != %v", len(evt.Tags), len(canonical.Tags))
		isBad = true
	}
	for i := range evt.Tags {
		if len(evt.Tags[i]) != len(canonical.Tags[i]) {
			t.Fatalf("tag[%d] length is wrong: `%v` != `%v`", i, len(evt.Tags[i]), len(canonical.Tags[i]))
			isBad = true
		}
		for j := range evt.Tags[i] {
			if evt.Tags[i][j] != canonical.Tags[i][j] {
				t.Fatalf("tag[%d][%d] is wrong: `%s` != `%s`", i, j, evt.Tags[i][j], canonical.Tags[i][j])
				isBad = true
			}
		}
	}

	return isBad
}

var nsonTestEvents = []string{
	`{"id":"192eaf31bd20476bbe9265a3667cfef6410dfd563c02a64cb15d6fa8efec0ed6","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","sig":"5b9051596a5ba0619fd5fd7d2766b8aeb0cc398f1d1a0804f4b4ed884482025b3d4888e4c892f2fc437415bfc121482a990fad30f5cd9e333e55364052f99bbc","created_at":1688505641,"nson":"0401000500","kind":1,"content":"hello","tags":[]}`,
	`{"id":"921ada34fe581b506975c641f2d1a3fb4f491f1d30c2490452e8524776895ebf","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","sig":"1f15a39e93a13f14f783eb127b2977e5dc5d207070dfa280fe45879b6b142ec1943ec921ab4268e69a43704d5641b45d18bf3789037c4842e062cd347a8a7ee1","created_at":1688553190,"nson":"12010006020200060005040005004000120006","kind":1,"content":"maçã","tags":[["entity","fruit"],["owner","79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","wss://リレー.jp","person"]]}`,
	`{"id":"06212bae3cfc917d4b1239a3bad4fdba1e0e1ff09fbd2ee7b6da15d5fd859f58","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","sig":"47199a3a4184528d2c6cbb94df03b9793ea65b4578154ff5edce794d03ee2408cd3ca699b39cc11e791656e98b510194330d3dc215389c5648eddf33b8362444","created_at":1688572619,"nson":"0401000400","kind":1,"content":"x\ny","tags":[]}`,
	`{"id":"ec9345e2af4225aada296964fa6025a1666dcac8dba154f5591a81f7dee1f84a","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","sig":"49f4b9edd7eff9e127b70077daff9a66da8c1ad974e5e6f47c094e8cc0c553071ff61c07b69d3db80c25f36248237ba6021038f5eb6b569ce79e3b024e8e358d","created_at":1688572819,"nson":"0401000400","kind":1,"content":"x\ty","tags":[]}`,
}

func BenchmarkNSONEncoding(b *testing.B) {
	events := make([]*nostr.Event, len(test_common.NormalEvents))
	for i, jevt := range test_common.NormalEvents {
		evt := &nostr.Event{}
		json.Unmarshal([]byte(jevt), evt)
		events[i] = evt
	}

	b.Run("easyjson.Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, evt := range events {
				easyjson.Marshal(evt)
			}
		}
	})

	b.Run("nson.Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, evt := range events {
				Marshal(evt)
			}
		}
	})
}

func BenchmarkNSONDecoding(b *testing.B) {
	events := make([]string, len(test_common.NormalEvents))
	for i, jevt := range test_common.NormalEvents {
		evt := &nostr.Event{}
		json.Unmarshal([]byte(jevt), evt)
		nevt, _ := Marshal(evt)
		events[i] = nevt
	}

	b.Run("easyjson.Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, nevt := range events {
				evt := &nostr.Event{}
				err := easyjson.Unmarshal([]byte(nevt), evt)
				if err != nil {
					b.Fatalf("failed to unmarshal: %s", err)
				}
			}
		}
	})

	b.Run("nson.Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, nevt := range events {
				evt := &nostr.Event{}
				err := Unmarshal(nevt, evt)
				if err != nil {
					b.Fatalf("failed to unmarshal: %s", err)
				}
			}
		}
	})

	b.Run("easyjson.Unmarshal+sig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, nevt := range events {
				evt := &nostr.Event{}
				err := easyjson.Unmarshal([]byte(nevt), evt)
				if err != nil {
					b.Fatalf("failed to unmarshal: %s", err)
				}
				evt.CheckSignature()
			}
		}
	})

	b.Run("nson.Unmarshal+sig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, nevt := range events {
				evt := &nostr.Event{}
				err := Unmarshal(nevt, evt)
				if err != nil {
					b.Fatalf("failed to unmarshal: %s", err)
				}
				evt.CheckSignature()
			}
		}
	})
}
