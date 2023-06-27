package benchmarks

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/nbd-wtf/go-nostr"
)

var nsonTestEvents = []string{
	`{"id":"ae1fc7154296569d87ca4663f6bdf448c217d1590d28c85d158557b8b43b4d69","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","sig":"94e10947814b1ebe38af42300ecd90c7642763896c4f69506ae97bfdf54eec3c0c21df96b7d95daa74ff3d414b1d758ee95fc258125deebc31df0c6ba9396a51","created_at":1683660344,"nson":"1405000b0203000100400005040001004000000014","kind":30023,"content":"hello hello","tags":[["e","b6de44a9dd47d1c000f795ea0453046914f44ba7d5e369608b04867a575ea83e","reply"],["p","c26f7b252cea77a5b94f42b1a4771021be07d4df766407e47738605f7e3ab774","","wss://relay.damus.io"]]}`,
	`{"id":"ae1fc7154296569d87ca4663f6bdf448c217d1590d28c85d158557b8b43b4d69","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","sig":"94e10947814b1ebe38af42300ecd90c7642763896c4f69506ae97bfdf54eec3c0c21df96b7d95daa74ff3d414b1d758ee95fc258125deebc31df0c6ba9396a51","created_at":1683660344,"nson":"140500100203000100400005040001004000000014","kind":30023,"content":"hello\n\"hello\"","tags":[["e","b6de44a9dd47d1c000f795ea0453046914f44ba7d5e369608b04867a575ea83e","reply"],["p","c26f7b252cea77a5b94f42b1a4771021be07d4df766407e47738605f7e3ab774","","wss://relay.damus.io"]]}`,
	`{"id":"a235323ad6ae7032667330c4d52def2d6be67a973d71f2f1784b2b5b01d57026","pubkey":"69aeace80672c08ef7729a03e597ed4e9dd5ddaa7c457349d55d12c043e8a7ab","sig":"7e5fedc3c1c16abb95d207b73a689b5f17ab039cffd0b6bea62dcbfb607a27d38c830542df5ce762685a0da4e8edd28beab0c9a8f47e3037ff6e676ea6297bfa","created_at":1680277541,"nson":"0401049600","kind":1,"content":"Hello #Plebstrs 🤙\n\nExcited to share with you the latest improvements we've designed to enhance your user experience when creating new posts on #Plebstr.\n\nMain UX improvements include:\n— The ability to mention anyone directly in your new post 🔍\n— Real-time previews of all attachments while creating your post \U0001fa7b\n— Minimizing the new post window to enable #multitasking and easy access to your draft 📝\n\nThis is the first design mockup and we can't wait to bring it to you in an upcoming updates. Our amazing developers are currently working on it, together with other small improvements. Stay tuned! 🚀👨\u200d💻\n\n*Some details may change so keep your fingers crossed 😄🤞\n\n#comingsoon #maybe #insomeshapeorform\n\nhttps://nostr.build/i/nostr.build_2b337d24f0cd19eff0678893ac93d58ee374ca8c3b9215516aa76a05856ec9c0.png\nhttps://nostr.build/i/nostr.build_f79b14687a1f6e7275192554722a85be815219573d824e381c4913715644e10d.png\nhttps://nostr.build/i/nostr.build_009fa068a32383f88e19763fa22e16142cf44166cb097293f7082e7bf4a38eed.png\nhttps://nostr.build/i/nostr.build_47ecdd4867808b3f3af2620ac4bf40aefcc9022c4ba26762567012c22d636487.png","tags":[]}}`,
}

func TestBasicNsonParse(t *testing.T) {
	for _, jevt := range nsonTestEvents {
		evt, _ := decodeNson(jevt)
		checkParsedCorrectly(t, evt, jevt)
	}
}

func TestNsonPartialGet(t *testing.T) {
	for _, jevt := range nsonTestEvents {
		evt, _ := decodeNson(jevt)

		if id := nsonGetID(jevt); id != evt.ID {
			t.Error("partial id wrong")
		}
		if pubkey := nsonGetPubkey(jevt); pubkey != evt.PubKey {
			t.Error("partial pubkey wrong")
		}
		if sig := nsonGetSig(jevt); sig != evt.Sig {
			t.Error("partial sig wrong")
		}
		if createdAt := nsonGetCreatedAt(jevt); createdAt != evt.CreatedAt {
			t.Error("partial created_at wrong")
		}
		if kind := nsonGetKind(jevt); kind != evt.Kind {
			t.Error("partial kind wrong")
		}
		if content := nsonGetContent(jevt); content != evt.Content {
			t.Error("partial content wrong")
		}
	}
}

func TestEncodeNson(t *testing.T) {
	jevt := `{
  "content": "hello world",
  "created_at": 1683762317,
  "id": "57ff66490a6a2af3992accc26ae95f3f60c6e5f84ed0ddf6f59c534d3920d3d2",
  "kind": 1,
  "pubkey": "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
  "sig": "504d142aed7fa7e0f6dab5bcd7eed63963b0277a8e11bbcb03b94531beb4b95a12f1438668b02746bd5362161bc782068e6b71494060975414e793f9e19f57ea",
  "tags": [
    [
      "e",
      "b6de44a9dd47d1c000f795ea0453046914f44ba7d5e369608b04867a575ea83e",
      "reply"
    ],
    [
      "p",
      "c26f7b252cea77a5b94f42b1a4771021be07d4df766407e47738605f7e3ab774",
      "",
      "wss://relay.damus.io"
    ]
  ]
}`

	evt := &nostr.Event{}
	json.Unmarshal([]byte(jevt), evt)

	nevt, _ := encodeNson(evt)
	fmt.Println(nevt)
}

func checkParsedCorrectly(t *testing.T, evt *nostr.Event, jevt string) (isBad bool) {
	var canonical nostr.Event
	err := json.Unmarshal([]byte(jevt), &canonical)
	fmt.Println(err)

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
	if evt.CreatedAt != nostr.Timestamp(canonical.CreatedAt) {
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
