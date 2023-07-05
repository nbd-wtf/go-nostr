package nson

import (
	"encoding/json"
	"testing"

	"github.com/nbd-wtf/go-nostr"
)

func TestBasicNsonParse(t *testing.T) {
	for _, jevt := range nsonTestEvents {
		evt := &nostr.Event{}
		if err := Unmarshal(jevt, evt); err != nil {
			t.Fatalf("error unmarshaling nson: %s", err)
		}
		checkParsedCorrectly(t, evt, jevt)
	}
}

func TestNsonPartialGet(t *testing.T) {
	for _, jevt := range nsonTestEvents {
		evt := &nostr.Event{}
		if err := Unmarshal(jevt, evt); err != nil {
			t.Fatalf("error unmarshaling nson: %s", err)
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
	for _, jevt := range normalEvents {
		pevt := &nostr.Event{}
		if err := json.Unmarshal([]byte(jevt), pevt); err != nil {
			t.Fatalf("failed to decode normal json: %s", err)
		}
		nevt, err := Marshal(*pevt)
		if err != nil {
			t.Fatalf("failed to encode nson: %s", err)
		}

		evt := &nostr.Event{}
		if err := Unmarshal(nevt, evt); err != nil {
			t.Fatalf("error unmarshaling nson: %s", err)
		}
		checkParsedCorrectly(t, pevt, jevt)
		checkParsedCorrectly(t, evt, jevt)
	}
}

func checkParsedCorrectly(t *testing.T, evt *nostr.Event, jevt string) (isBad bool) {
	var canonical nostr.Event
	err := json.Unmarshal([]byte(jevt), &canonical)
	if err != nil {
		t.Fatalf("error unmarshaling normal json: %s", err)
		return
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

var normalEvents = []string{
	`{"id":"99b83b56b5e32d41bb950b53e68c8b9e25cb2c5aad0a91f5a063e1899cd610d7","pubkey":"5ec2d2c42dda8b0a560a145f6ef2eae3be8f9f972ca33aca6720de96572f12b9","created_at":1688572804,"kind":1,"tags":[],"content":"Time: 05/07/23 12:00:03\nUptime: 6 days, 17:50:45\n\nCPU:\n\tUsage: 9.8%\n\tTemperature: 35.67°C\nCore Temps:\n\tCore 0:\t35°C\n\tCore 1:\t36°C\n\tCore 2:\t34°C\n\tCore 3:\t35°C\n\tCore 4:\t39°C\n\tCore 5:\t35°C\n\nMemory:\n\tTotal: 15.57 GB\n\tUsed: 3.45 GB\n\tPercent Used: 24.3%","sig":"9eff509ed6fc96067ddee9a7d6c7abfe066136c7d82f0b3601c956765402efa9591e6916ca35aa06976c23b1adb2d368bd0f8d21d73e5f7c74d58acd1599c73a"}`,
	`{"id":"080c1acd1df07693fd59ad205d14c4d966a1729c6c6773e2b131f5d2356ace77","pubkey":"06a498e5bf0cd756a4941e422713a7e75deca00332cb3736000f3df8616a2367","created_at":1688556260,"kind":30078,"tags":[["d","plebstr"]],"content":"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?iv=ILkIKiNWJ1xZhh69TAiCOQ==","sig":"31e6b022f1b7133a97490faebeb75f08ba230100df36ad11440bb8547c83cb42d741d8fc2bfee7880f33e864d354092532fe4a9b6191245a01ff65ea00f244c1"}`,
	`{"id":"55ef38277352859c9e70a70e17e565652d5ece390ef05225104bf6f846410f0f","pubkey":"e81ca829c9bd368cc584844078f570c105e59d9392d19ce71bb9f34c1ac633f3","created_at":1688556088,"kind":1,"tags":[["e","29d57dd3bff6fde72141efcf55a09da0e4cb4a41785aa4f7c1411f8505af72b7","","reply"],["p","1e2d080673f959a5d82357d5e2aa5011778af634c33e4207cc54e7df943c798c"]],"content":"Is today the opportunity?","sig":"e9575aa169dbe38c249d7fedae70d1bed9bebca8522793a3d98ab2a12ef3849f85c87a3af2f24557296ef049f7b1f5ff09c5a1d812487ab26fa669d0093840bb"}`,
	`{"id":"221e4c29c3ea93ddcd2298aaf5a0f5a7c628afb79d005cbb415cef2af8a2bb77","pubkey":"e81ca829c9bd368cc584844078f570c105e59d9392d19ce71bb9f34c1ac633f3","created_at":1688556080,"kind":6,"tags":[["e","29d57dd3bff6fde72141efcf55a09da0e4cb4a41785aa4f7c1411f8505af72b7"],["p","1e2d080673f959a5d82357d5e2aa5011778af634c33e4207cc54e7df943c798c"]],"content":"{\"content\":\"There will always be a another opportunity to buy more Bitcoin. On our way to 1 Whole Bitcoin…. #bitcoin #dip #nostr #plebchain\\n\\n\\n\\nhttps://nostrcheck.me/media/public/nostrcheck.me_2617026328114791421688555844.webp \",\"created_at\":1688555863,\"id\":\"29d57dd3bff6fde72141efcf55a09da0e4cb4a41785aa4f7c1411f8505af72b7\",\"kind\":1,\"pubkey\":\"1e2d080673f959a5d82357d5e2aa5011778af634c33e4207cc54e7df943c798c\",\"sig\":\"5d60fad4103a82934b9fde378b36b67db811b624da70c57f5ff1b50a11e0d606de606e1593a2d7446ed7ab2fc56bb13d89280f9336f6a74c40eb98f9d274bd81\",\"tags\":[[\"t\",\"bitcoin\"],[\"t\",\"dip\"],[\"t\",\"nostr\"],[\"t\",\"plebchain\"]]}","sig":"6cbeaae55176f424520cb13bfa5287e67438b3950653159c914bf7ce838097c29a4e3b95f84610cc8d211b5dc76872482b9cd0cfe09ba5bc84eae71d974a30a9"}`,
	`{"id":"2dc1a37fce7815aba8a1750801f86c1cd35145bba6cfc35cce2c9c96eef32e5f","pubkey":"7ca66d4166b16f54a16868191ba1c6386a976624f4634f3896d9b6740a388ca3","created_at":1688556074,"kind":1,"tags":[["q","d913924e45928baf48b6b8fce440ebb7ccd177bc0979350923f5375aa42ceda6"],["imeta","url https://nostr.build/av/43715004b4a8ab944a45160869b9f01b1733f453817b4aacf938f563142aa735.mov","blurhash eaDv1LD*ICtkxV}uNGO9nmniVvt5ovaOWCEz$jw1XQX8Ioxas.R*jb","dim 720x1280"]],"content":"Lord knows their Magic 8 Ball is useless https://nostr.build/av/43715004b4a8ab944a45160869b9f01b1733f453817b4aacf938f563142aa735.mov  nostr:note1myfeynj9j2967j9khr7wgs8tklxdzaaup9un2zfr75m44fpvaknq0qhsgt","sig":"e3d6f7d2deea211299f22d97be779629f66c31ee6a84382e04503817f2ecf16dacdfe4f535da378b508f917c6c73bcfb32b60510ed4edca32f7247fae4ae7ff6"}`,
	`{"id":"989a336e2b5f35080afa97b72bfe88f42381c9e624d1849417f364e06b2221b0","pubkey":"634bd19e5c87db216555c814bf88e66ace175805291a6be90b15ac3b2247da9b","created_at":1688557054,"kind":1,"tags":[],"content":"あーあーあーあー、てすてす","sig":"d1ab7eeb73779f2a5bb6a3339aa5afb16afd3347b663823f135f5343c2eea9a4e337565f97e7a4dac34bf75f227489a27f3321fd740c1a426968fb5a76c99717"}`,
	`{"id":"0d6cf58fe2878c050973bb26e678090258c716c456008aa6d849de555fa788b3","pubkey":"e472cba86ba9df4a48605371a42e90117036cbc1f9919865809346e59064b28f","created_at":1688557024,"kind":1,"tags":[],"content":"strfryのstreamとsyncの違いか今の所よくわからない…","sig":"e7de14d5b6f62c44c3f24838d23e388feabaf2500144e5ca2630adf34bc4e7f512c4f7303109ba9fd4c803d47bd8a48bdacc2e29aa1701c8c6dbfbf3dc9240da"}`,
	`{"id":"c290be21ddf6188436bf544d5625246de2dde22eb17ab41f40b6b8aa9bee9c98","pubkey":"4d39c23b3b03bf99494df5f3a149c7908ae1bc7416807fdd6b34a31886eaae25","created_at":1688556176,"kind":1,"tags":[],"content":"独裁かはわからんけど、ぽーまんさんはキャッチミーイフユーキャンの詐欺師みたいな感じ","sig":"b8e10a7df4718f0738c0bbc59b7f25401027fa436dc00f0afdcb979bd253050376bbaea1a6ec5fa246be935d6cd5f72d8010e8f800c79a9867f00f5b1e083a14"}`,
	`{"id":"0ad438f0a34756ecb1bf4d1792dc42a5b0141a39d944dfdd6737e883815a65dc","pubkey":"6a3cdfe891cddc33228a52cd7b27eca17e630569c93c24d70dc1cc01ce45881f","created_at":1688556173,"kind":1,"tags":[],"content":"hallucination やめて","sig":"ddbcc08b16f88532ccc739ab7dfa112fb462aafbeecb859a1b1b511ae9c2eb46872505aec58fe7e8b38639e558f0e9e0a13adf1b2f89d3a96f890acb3cd5c40f"}`,
	`{"id":"ef1aea4c78f3de5cdd07dfe632e83adef34b3ac0c26afba60852ecd9800adc16","pubkey":"634bd19e5c87db216555c814bf88e66ace175805291a6be90b15ac3b2247da9b","created_at":1688556039,"kind":1,"tags":[],"content":"※日本リレーの relay-jp.nostr.wirednet.jp は何もいじらないので、継続して利用可能です","sig":"94eba6e0a242cf8987e1d8d782968b9e341e4f66278b937fa4da33c708e1f6eb82652796785eb20b21f9c18c0534a568b088297b6bef65729192ea04485b7740"}`,
	`{"id":"d2c2cee862a4c7c903ecaf129e2458132b3b4134ae3135f71ba4b84798ccdd3f","pubkey":"634bd19e5c87db216555c814bf88e66ace175805291a6be90b15ac3b2247da9b","created_at":1688555969,"kind":1,"tags":[],"content":"relay.nostr.wirednet.jp をちょっとメンテナンスしますー\n一時的に過去のデータにはアクセス出来なくなります。(そのうち復活させる)","sig":"9c0749183db90cac31778523424453ba53532f7537233053fb1629428a4844bc9e69efdb2a2ac75b3e6f10fd28a34c366d79fa86f68a3fba36fea2bcd82d5c9f"}`,
	`{"id":"4296bfa40427b9cb3e078da9c12de7af57e238caf77ace9b517ecd99ad7f38d8","pubkey":"046284c5d3cc859f58b1ff58d2bdbf22eb6f41a633e97f503a569cc1fe886322","created_at":1688555517,"kind":1,"tags":[],"content":"ブンブンピーブピー","sig":"40426c3677dd61132558e58ec2e0d306a7581a73e7cbcd8fcf447b0da1580b782c12461d4105939faa4caf95864354dba25fe5b10aa794ccc7f68adb2d12bb01"}`,
	`{"id":"abd1d0c9300b7745bfada6147ceb5b4d9d09ab23925e55c53b835347fdd0cb17","pubkey":"634bd19e5c87db216555c814bf88e66ace175805291a6be90b15ac3b2247da9b","created_at":1688554980,"kind":1,"tags":[],"content":"Threadsには旅立たないかなー。","sig":"4f0243d5380a1757d78a772bb27386d2c2b54926b514f4568e717ed9cfe6d87f8d299a9b34d6bbd90241deabde17a3bf514f3195b4f4c4183429387bdc6f179d"}`,
	`{"id":"4db06f7e522db1d5166f5455e193690a3e79f256ffa27df09aeede7f70fd87f1","pubkey":"2748ffc20bf0378ace2b32d4e9ca11fceb07fbef335a7124b5368b6338daf18b","created_at":1688554800,"kind":1,"tags":[],"content":"ζ*'ヮ')ζ＜ｳｯｳｰ!! 🕗\n--------\n2023年 7月5日 (水)　20：00：00","sig":"16aa8eda88e42711cd2b77f5611cb0f171493d36d58c167513afa3be3bbfb3f3ddc7cbb6a20a8b8e011c0b61befbc8d6e8b012f49619f9a15d77410e849df185"}`,
	`{"id":"ebd8dd36f274ddf91959bf1225bb4c0353d187b373d91e92e1f971365d556420","pubkey":"634bd19e5c87db216555c814bf88e66ace175805291a6be90b15ac3b2247da9b","created_at":1688554184,"kind":1,"tags":[],"content":"あーーーーーーー、 relay.nostr.wirednet.jp ののぞき窓。\nログインボタンを非表示にしてるんだけど、キーボードショートカットだけ生きてることに気付いた。(※作者です)","sig":"21fc8e74b995bd185031fac03b85e3a1b431f79de26658f00c50e404769ac431ca53f151c3dc9a90435b2e70f4a2ac199c84fcb7ca2858c45665d99f9f9bae0a"}`,
	`{"id":"e2aec1b7e297329203f67b61f214c2b745a3bc1590f299ca250a1633714c829c","pubkey":"b6ac413652c8431478cb6177722f822f0f7af774a274fc5574872407834c3253","created_at":1688553478,"kind":1,"tags":[],"content":"やー今日も疲れたなー！\n大将！お勧めでイソシアネートとポリオールね！\nあ、6:4でよろしく！","sig":"12ba5dc9ff18f4ce995941f6de3bfaf8e3636afde37a06a4d3478c930ae22e2f79690e6f0682d532541222746aeb5f6dda29251cd7c31e71d7e206199b04bab4"}`,
	`{"id":"e4e86256ed64514bcb3350cf8b631ef84b4aeafcdb164cea5096c893ead6a0a1","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","created_at":1688574304,"kind":1,"tags":[],"content":"\b\f\ueeee","sig":"c61a4971facc4899109e1a28b73cbd27f8807fedcff87cfa1d8f5e9b709feab75e3a62a96fc75b5d2a2f42443d5ca35daa6c3d724cd6e6133b9c4a1ef072c1e9"}`,
}
