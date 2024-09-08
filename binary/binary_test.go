package binary

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/test_common"
	"github.com/stretchr/testify/require"
)

func TestBinaryPartialGet(t *testing.T) {
	for _, jevt := range test_common.NormalEvents {
		evt := &nostr.Event{}
		json.Unmarshal([]byte(jevt), &evt)
		bevt, err := Marshal(evt)
		if err != nil {
			t.Fatalf("error marshalling binary: %s", err)
		}

		if id := hex.EncodeToString(bevt[0:32]); id != evt.ID {
			t.Fatalf("partial id wrong. got %v, expected %v", id, evt.ID)
		}
		if pubkey := hex.EncodeToString(bevt[32:64]); pubkey != evt.PubKey {
			t.Fatalf("partial pubkey wrong. got %v, expected %v", pubkey, evt.PubKey)
		}
		if sig := hex.EncodeToString(bevt[64:128]); sig != evt.Sig {
			t.Fatalf("partial sig wrong. got %v, expected %v", sig, evt.Sig)
		}
		if createdAt := nostr.Timestamp(binary.BigEndian.Uint32(bevt[128:132])); createdAt != evt.CreatedAt {
			t.Fatalf("partial created_at wrong. got %v, expected %v", createdAt, evt.CreatedAt)
		}
		if kind := int(binary.BigEndian.Uint16(bevt[132:134])); kind != evt.Kind {
			t.Fatalf("partial kind wrong. got %v, expected %v", kind, evt.Kind)
		}
		if content := string(bevt[136 : 136+int(binary.BigEndian.Uint16(bevt[134:136]))]); content != evt.Content {
			t.Fatalf("partial content wrong. got %v, expected %v", content, evt.Content)
		}
	}
}

func TestBinaryEncodeBackwardsCompatible(t *testing.T) {
	for i, jevt := range test_common.NormalEvents {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			b64bevt := test_common.BinaryEventsBase64[i]
			bevt, err := base64.StdEncoding.DecodeString(b64bevt)
			require.NoError(t, err)

			pevt := &nostr.Event{}
			err = json.Unmarshal([]byte(jevt), pevt)
			require.NoError(t, err)

			encoded, err := Marshal(pevt)
			require.NoError(t, err)

			require.Equal(t, bevt, encoded)
		})
	}
}

func TestBinaryEncode(t *testing.T) {
	for i, jevt := range test_common.NormalEvents {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			pevt := &nostr.Event{}
			if err := json.Unmarshal([]byte(jevt), pevt); err != nil {
				t.Fatalf("failed to decode normal json: %s", err)
			}
			bevt, err := Marshal(pevt)
			if err != nil {
				t.Fatalf("failed to encode binary: %s", err)
			}
			evt := &nostr.Event{}
			if err := Unmarshal(bevt, evt); err != nil {
				t.Fatalf("error unmarshalling binary: %s", err)
			}

			checkParsedCorrectly(t, pevt, jevt)
			checkParsedCorrectly(t, evt, jevt)
		})
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
