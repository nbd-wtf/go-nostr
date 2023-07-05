package nson

import (
	"encoding/json"
	"testing"

	"github.com/nbd-wtf/go-nostr"
)

var nsonTestEvents = []string{
	`{"id":"192eaf31bd20476bbe9265a3667cfef6410dfd563c02a64cb15d6fa8efec0ed6","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","sig":"5b9051596a5ba0619fd5fd7d2766b8aeb0cc398f1d1a0804f4b4ed884482025b3d4888e4c892f2fc437415bfc121482a990fad30f5cd9e333e55364052f99bbc","created_at":1688505641,"nson":"0401000500","kind":1,"content":"hello","tags":[]}`,
	`{"id":"921ada34fe581b506975c641f2d1a3fb4f491f1d30c2490452e8524776895ebf","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","sig":"1f15a39e93a13f14f783eb127b2977e5dc5d207070dfa280fe45879b6b142ec1943ec921ab4268e69a43704d5641b45d18bf3789037c4842e062cd347a8a7ee1","created_at":1688553190,"nson":"12010006020200060005040005004000120006","kind":1,"content":"maçã","tags":[["entity","fruit"],["owner","79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","wss://リレー.jp","person"]]}`,
	`{"id":"7dfb54d7c7283d4710195d46228fa495f0240f65e000a159cbe6110673b0d1a5","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","sig":"2d4131c1ab9eb5a03cf8e47387c5b6a79edde9909eae72507e88a213cf2859f82d0654e3d15a60dd1d9bf5343e1e06e38cae04c3c379a5920750717decb4bea1","created_at":1688556395,"nson":"0401000600","kind":1,"content":"x\\n\\","tags":[]}`,
}

func TestBasicNsonParse(t *testing.T) {
	for _, jevt := range nsonTestEvents {
		evt, err := Unmarshal(jevt)
		if err != nil {
			t.Errorf("error unmarshaling nson: %s", err)
		}
		checkParsedCorrectly(t, evt, jevt)
	}
}

func TestNsonPartialGet(t *testing.T) {
	for _, jevt := range nsonTestEvents {
		evt, err := Unmarshal(jevt)
		if err != nil {
			t.Errorf("error unmarshaling nson: %s", err)
		}

		wrapper := New(jevt)

		if id := wrapper.GetID(); id != evt.ID {
			t.Errorf("partial id wrong. got %v, expected %v", id, evt.ID)
		}
		if pubkey := wrapper.GetPubkey(); pubkey != evt.PubKey {
			t.Errorf("partial pubkey wrong. got %v, expected %v", pubkey, evt.PubKey)
		}
		if sig := wrapper.GetSig(); sig != evt.Sig {
			t.Errorf("partial sig wrong. got %v, expected %v", sig, evt.Sig)
		}
		if createdAt := wrapper.GetCreatedAt(); createdAt != evt.CreatedAt {
			t.Errorf("partial created_at wrong. got %v, expected %v", createdAt, evt.CreatedAt)
		}
		if kind := wrapper.GetKind(); kind != evt.Kind {
			t.Errorf("partial kind wrong. got %v, expected %v", kind, evt.Kind)
		}
		if content := wrapper.GetContent(); content != evt.Content {
			t.Errorf("partial content wrong. got %v, expected %v", content, evt.Content)
		}
	}
}

func checkParsedCorrectly(t *testing.T, evt *nostr.Event, jevt string) (isBad bool) {
	var canonical nostr.Event
	err := json.Unmarshal([]byte(jevt), &canonical)
	if err != nil {
		t.Errorf("error unmarshaling normal json: %s", err)
		return
	}

	if evt.ID != canonical.ID {
		t.Errorf("id is wrong: %s != %s", evt.ID, canonical.ID)
		isBad = true
	}
	if evt.PubKey != canonical.PubKey {
		t.Errorf("pubkey is wrong: %s != %s", evt.PubKey, canonical.PubKey)
		isBad = true
	}
	if evt.Sig != canonical.Sig {
		t.Errorf("sig is wrong: %s != %s", evt.Sig, canonical.Sig)
		isBad = true
	}
	if evt.Content != canonical.Content {
		t.Errorf("content is wrong: %s != %s", evt.Content, canonical.Content)
		isBad = true
	}
	if evt.Kind != canonical.Kind {
		t.Errorf("kind is wrong: %d != %d", evt.Kind, canonical.Kind)
		isBad = true
	}
	if evt.CreatedAt != canonical.CreatedAt {
		t.Errorf("created_at is wrong: %v != %v", evt.CreatedAt, canonical.CreatedAt)
		isBad = true
	}
	if len(evt.Tags) != len(canonical.Tags) {
		t.Errorf("tag number is wrong: %v != %v", len(evt.Tags), len(canonical.Tags))
		isBad = true
	}
	for i := range evt.Tags {
		if len(evt.Tags[i]) != len(canonical.Tags[i]) {
			t.Errorf("tag[%d] length is wrong: `%v` != `%v`", i, len(evt.Tags[i]), len(canonical.Tags[i]))
			isBad = true
		}
		for j := range evt.Tags[i] {
			if evt.Tags[i][j] != canonical.Tags[i][j] {
				t.Errorf("tag[%d][%d] is wrong: `%s` != `%s`", i, j, evt.Tags[i][j], canonical.Tags[i][j])
				isBad = true
			}
		}
	}

	return isBad
}
