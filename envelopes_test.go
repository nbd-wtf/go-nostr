//go:build sonic

package nostr

import (
	"bufio"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMessage(t *testing.T) {
	testCases := []struct {
		Name             string
		Message          string
		ExpectedEnvelope Envelope
	}{
		{
			Name:             "nil",
			Message:          "",
			ExpectedEnvelope: nil,
		},
		{
			Name:             "invalid string",
			Message:          "invalid input",
			ExpectedEnvelope: nil,
		},
		{
			Name:             "invalid string with a comma",
			Message:          "invalid, input",
			ExpectedEnvelope: nil,
		},
		{
			Name:             "EVENT envelope with subscription id",
			Message:          `["EVENT","_",{"kind":1,"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}]`,
			ExpectedEnvelope: &EventEnvelope{SubscriptionID: ptr("_"), Event: Event{Kind: 1, ID: "dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962", PubKey: "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d", CreatedAt: 1644271588, Tags: Tags{}, Content: "now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?", Sig: "230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}},
		},
		{
			Name:             "EVENT envelope without subscription id",
			Message:          `["EVENT",{"kind":1,"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}]`,
			ExpectedEnvelope: &EventEnvelope{Event: Event{Kind: 1, ID: "dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962", PubKey: "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d", CreatedAt: 1644271588, Tags: Tags{}, Content: "now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?", Sig: "230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}},
		},
		{
			Name:             "EVENT envelope with tags",
			Message:          `["EVENT",{"kind":3,"id":"9e662bdd7d8abc40b5b15ee1ff5e9320efc87e9274d8d440c58e6eed2dddfbe2","pubkey":"373ebe3d45ec91977296a178d9f19f326c70631d2a1b0bbba5c5ecc2eb53b9e7","created_at":1644844224,"tags":[["p","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"],["e","75fc5ac2487363293bd27fb0d14fb966477d0f1dbc6361d37806a6a740eda91e"],["p","46d0dfd3a724a302ca9175163bdf788f3606b3fd1bb12d5fe055d1e418cb60ea"]],"content":"{\"wss://nostr-pub.wellorder.net\":{\"read\":true,\"write\":true},\"wss://nostr.bitcoiner.social\":{\"read\":false,\"write\":true},\"wss://expensive-relay.fiatjaf.com\":{\"read\":true,\"write\":true},\"wss://relayer.fiatjaf.com\":{\"read\":true,\"write\":true},\"wss://relay.bitid.nz\":{\"read\":true,\"write\":true},\"wss://nostr.rocks\":{\"read\":true,\"write\":true}}","sig":"811355d3484d375df47581cb5d66bed05002c2978894098304f20b595e571b7e01b2efd906c5650080ffe49cf1c62b36715698e9d88b9e8be43029a2f3fa66be"}]`,
			ExpectedEnvelope: &EventEnvelope{Event: Event{Kind: 3, ID: "9e662bdd7d8abc40b5b15ee1ff5e9320efc87e9274d8d440c58e6eed2dddfbe2", PubKey: "373ebe3d45ec91977296a178d9f19f326c70631d2a1b0bbba5c5ecc2eb53b9e7", CreatedAt: 1644844224, Tags: Tags{Tag{"p", "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"}, Tag{"e", "75fc5ac2487363293bd27fb0d14fb966477d0f1dbc6361d37806a6a740eda91e"}, Tag{"p", "46d0dfd3a724a302ca9175163bdf788f3606b3fd1bb12d5fe055d1e418cb60ea"}}, Content: "{\"wss://nostr-pub.wellorder.net\":{\"read\":true,\"write\":true},\"wss://nostr.bitcoiner.social\":{\"read\":false,\"write\":true},\"wss://expensive-relay.fiatjaf.com\":{\"read\":true,\"write\":true},\"wss://relayer.fiatjaf.com\":{\"read\":true,\"write\":true},\"wss://relay.bitid.nz\":{\"read\":true,\"write\":true},\"wss://nostr.rocks\":{\"read\":true,\"write\":true}}", Sig: "811355d3484d375df47581cb5d66bed05002c2978894098304f20b595e571b7e01b2efd906c5650080ffe49cf1c62b36715698e9d88b9e8be43029a2f3fa66be"}},
		},
		{
			Name:             "NOTICE envelope",
			Message:          `["NOTICE","kjasbdlasvdluiasvd\"kjasbdksab\\d"]`,
			ExpectedEnvelope: ptr(NoticeEnvelope("kjasbdlasvdluiasvd\"kjasbdksab\\d")),
		},
		{
			Name:             "EOSE envelope",
			Message:          `["EOSE","kjasbdlasvdluiasvd\"kjasbdksab\\d"]`,
			ExpectedEnvelope: ptr(EOSEEnvelope("kjasbdlasvdluiasvd\"kjasbdksab\\d")),
		},
		{
			Name:             "COUNT envelope",
			Message:          `["COUNT","z",{"count":12}]`,
			ExpectedEnvelope: &CountEnvelope{SubscriptionID: "z", Count: ptr(int64(12))},
		},
		{
			Name:             "COUNT envelope with HLL",
			Message:          `["COUNT","sub1",{"count":42, "hll": "0100000101000000000000040000000001020000000002000000000200000003000002040000000101020001010000000000000007000004010000000200040000020400000000000102000002000004010000010000000301000102030002000301000300010000070000000001000004000102010000000400010002000000000103000100010001000001040100020001000000000000010000020000000000030100000001000400010000000000000901010100000000040000000b030000010100010000010000010000000003000000000000010003000100020000000000010000010100000100000104000200030001000300000001000101000102"}]`,
			ExpectedEnvelope: &CountEnvelope{SubscriptionID: "sub1", Count: ptr(int64(42)), HyperLogLog: []byte{1, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 1, 2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 2, 4, 0, 0, 0, 1, 1, 2, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 4, 1, 0, 0, 0, 2, 0, 4, 0, 0, 2, 4, 0, 0, 0, 0, 0, 1, 2, 0, 0, 2, 0, 0, 4, 1, 0, 0, 1, 0, 0, 0, 3, 1, 0, 1, 2, 3, 0, 2, 0, 3, 1, 0, 3, 0, 1, 0, 0, 7, 0, 0, 0, 0, 1, 0, 0, 4, 0, 1, 2, 1, 0, 0, 0, 4, 0, 1, 0, 2, 0, 0, 0, 0, 1, 3, 0, 1, 0, 1, 0, 1, 0, 0, 1, 4, 1, 0, 2, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 2, 0, 0, 0, 0, 0, 3, 1, 0, 0, 0, 1, 0, 4, 0, 1, 0, 0, 0, 0, 0, 0, 9, 1, 1, 1, 0, 0, 0, 0, 4, 0, 0, 0, 11, 3, 0, 0, 1, 1, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 1, 0, 3, 0, 1, 0, 2, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0, 1, 4, 0, 2, 0, 3, 0, 1, 0, 3, 0, 0, 0, 1, 0, 1, 1, 0, 1, 2}},
		},
		{
			Name:             "OK envelope success",
			Message:          `["OK","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefaaaaa",true,""]`,
			ExpectedEnvelope: &OKEnvelope{EventID: "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefaaaaa", OK: true, Reason: ""},
		},
		{
			Name:             "OK envelope failure",
			Message:          `["OK","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefaaaaa",false,"error: could not connect to the database"]`,
			ExpectedEnvelope: &OKEnvelope{EventID: "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefaaaaa", OK: false, Reason: "error: could not connect to the database"},
		},
		{
			Name:             "CLOSED envelope with underscore",
			Message:          `["CLOSED","_","error: something went wrong"]`,
			ExpectedEnvelope: &ClosedEnvelope{SubscriptionID: "_", Reason: "error: something went wrong"},
		},
		{
			Name:             "CLOSED envelope with colon",
			Message:          `["CLOSED",":1","auth-required: take a selfie and send it to the CIA"]`,
			ExpectedEnvelope: &ClosedEnvelope{SubscriptionID: ":1", Reason: "auth-required: take a selfie and send it to the CIA"},
		},
		{
			Name:             "AUTH envelope with challenge",
			Message:          `["AUTH","kjsabdlasb aslkd kasndkad \"as.kdnbskadb"]`,
			ExpectedEnvelope: &AuthEnvelope{Challenge: ptr("kjsabdlasb aslkd kasndkad \"as.kdnbskadb")},
		},
		{
			Name:             "AUTH envelope with event",
			Message:          `["AUTH",{"kind":1,"id":"ae1fc7154296569d87ca4663f6bdf448c217d1590d28c85d158557b8b43b4d69","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","created_at":1683660344,"tags":[],"content":"hello world","sig":"94e10947814b1ebe38af42300ecd90c7642763896c4f69506ae97bfdf54eec3c0c21df96b7d95daa74ff3d414b1d758ee95fc258125deebc31df0c6ba9396a51"}]`,
			ExpectedEnvelope: &AuthEnvelope{Event: Event{Kind: 1, ID: "ae1fc7154296569d87ca4663f6bdf448c217d1590d28c85d158557b8b43b4d69", PubKey: "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798", CreatedAt: 1683660344, Tags: Tags{}, Content: "hello world", Sig: "94e10947814b1ebe38af42300ecd90c7642763896c4f69506ae97bfdf54eec3c0c21df96b7d95daa74ff3d414b1d758ee95fc258125deebc31df0c6ba9396a51"}},
		},
		{
			Name:             "REQ envelope",
			Message:          `["REQ","million", {"kinds": [1]}, {"kinds": [30023 ], "#d": ["buteko",    "batuke"]}]`,
			ExpectedEnvelope: &ReqEnvelope{SubscriptionID: "million", Filters: Filters{{Kinds: []int{1}}, {Kinds: []int{30023}, Tags: TagMap{"d": []string{"buteko", "batuke"}}}}},
		},
		{
			Name:             "CLOSE envelope",
			Message:          `["CLOSE","subscription123"]`,
			ExpectedEnvelope: ptr(CloseEnvelope("subscription123")),
		},
		{
			Name:             "AUTH envelope from nak 23gmt bug",
			Message:          `["AUTH","c45b2b06ad92e28a"]`,
			ExpectedEnvelope: &AuthEnvelope{Challenge: ptr("c45b2b06ad92e28a")},
		},
		{
			Name:             "REQ from jumble",
			Message:          `["REQ","sub:1",{"kinds":[1,6],"limit":100}]`,
			ExpectedEnvelope: &ReqEnvelope{SubscriptionID: "sub:1", Filters: Filters{{Kinds: []int{1, 6}, Limit: 100}}},
		},
	}

	t.Run("standard", func(t *testing.T) {
		for _, testCase := range testCases {
			t.Run(testCase.Name, func(t *testing.T) {
				envelope := ParseMessage(testCase.Message)
				if testCase.ExpectedEnvelope == nil && envelope == nil {
					return
				}

				if testCase.ExpectedEnvelope == nil {
					require.Nil(t, envelope, "expected nil but got %v", envelope)
					return
				}

				require.NotNil(t, envelope, "expected non-nil envelope but got nil")
				require.Equal(t, testCase.ExpectedEnvelope.String(), envelope.String())
			})
		}
	})

	t.Run("sonic", func(t *testing.T) {
		smp := NewSonicMessageParser()

		for _, testCase := range testCases {
			t.Run(testCase.Name, func(t *testing.T) {
				envelope, err := smp.ParseMessage(testCase.Message)

				if testCase.ExpectedEnvelope == nil && envelope == nil {
					return
				}

				if testCase.ExpectedEnvelope == nil {
					require.Nil(t, envelope, "expected nil but got %v", envelope)
					return
				}

				require.NoError(t, err)
				require.NotNil(t, envelope, "expected non-nil envelope but got nil")
				require.Equal(t, testCase.ExpectedEnvelope, envelope)
			})
		}
	})
}

func TestParseMessagesFromFile(t *testing.T) {
	file, err := os.Open("testdata/messages.jsonl")
	if err != nil {
		t.Skipf("Skipping test because testdata/messages.jsonl could not be opened: %v", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	smp := NewSonicMessageParser()
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		standardEnvelope := ParseMessage(string(line))
		sonicEnvelope, err := smp.ParseMessage(string(line))

		if standardEnvelope == nil {
			require.Nil(t, sonicEnvelope, "line %d: standard parser returned nil but sonic parser didn't", lineNum)
			continue
		}

		require.NoError(t, err, "line %d: sonic parser returned error", lineNum)
		require.NotNil(t, sonicEnvelope, "line %d: standard parser returned non-nil but sonic parser returned nil", lineNum)

		require.Equal(t, standardEnvelope, sonicEnvelope,
			"line %d: parsers returned different results", lineNum)
	}

	require.NoError(t, scanner.Err(), "error reading file")
}
