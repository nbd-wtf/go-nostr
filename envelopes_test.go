package nostr

import (
	"testing"

	"github.com/minio/simdjson-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventEnvelopeEncodingAndDecoding(t *testing.T) {
	eventEnvelopes := []string{
		`["EVENT","_",{"kind":1,"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}]`,
		`["EVENT",{"kind":3,"id":"9e662bdd7d8abc40b5b15ee1ff5e9320efc87e9274d8d440c58e6eed2dddfbe2","pubkey":"373ebe3d45ec91977296a178d9f19f326c70631d2a1b0bbba5c5ecc2eb53b9e7","created_at":1644844224,"tags":[["p","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"],["p","75fc5ac2487363293bd27fb0d14fb966477d0f1dbc6361d37806a6a740eda91e"],["p","46d0dfd3a724a302ca9175163bdf788f3606b3fd1bb12d5fe055d1e418cb60ea"]],"content":"{\"wss://nostr-pub.wellorder.net\":{\"read\":true,\"write\":true},\"wss://nostr.bitcoiner.social\":{\"read\":false,\"write\":true},\"wss://expensive-relay.fiatjaf.com\":{\"read\":true,\"write\":true},\"wss://relayer.fiatjaf.com\":{\"read\":true,\"write\":true},\"wss://relay.bitid.nz\":{\"read\":true,\"write\":true},\"wss://nostr.rocks\":{\"read\":true,\"write\":true}}","sig":"811355d3484d375df47581cb5d66bed05002c2978894098304f20b595e571b7e01b2efd906c5650080ffe49cf1c62b36715698e9d88b9e8be43029a2f3fa66be"}]`,
	}

	for _, raw := range eventEnvelopes {
		var env EventEnvelope
		err := json.Unmarshal([]byte(raw), &env)
		assert.NoError(t, err)
		assert.Equal(t, env.GetID(), env.ID)

		ok, _ := env.CheckSignature()
		assert.True(t, ok)

		asJSON, err := json.Marshal(env)
		assert.NoError(t, err)
		assert.Equal(t, raw, string(asJSON))
	}
}

func TestNoticeEnvelopeEncodingAndDecoding(t *testing.T) {
	noticeEnv := `["NOTICE","kjasbdlasvdluiasvd\"kjasbdksab\\d"]`
	var env NoticeEnvelope
	err := json.Unmarshal([]byte(noticeEnv), &env)
	assert.NoError(t, err)
	assert.Equal(t, "kjasbdlasvdluiasvd\"kjasbdksab\\d", string(env))

	res, err := json.Marshal(env)
	assert.NoError(t, err)
	assert.Equal(t, noticeEnv, string(res))
}

func TestEoseEnvelopeEncodingAndDecoding(t *testing.T) {
	eoseEnv := `["EOSE","kjasbdlasvdluiasvd\"kjasbdksab\\d"]`
	var env EOSEEnvelope
	err := json.Unmarshal([]byte(eoseEnv), &env)
	assert.NoError(t, err)
	assert.Equal(t, "kjasbdlasvdluiasvd\"kjasbdksab\\d", string(env))

	res, err := json.Marshal(env)
	assert.NoError(t, err)
	assert.Equal(t, eoseEnv, string(res))
}

func TestCountEnvelopeEncodingAndDecoding(t *testing.T) {
	countEnv := `["COUNT","z",{"count":12}]`
	var env CountEnvelope
	err := json.Unmarshal([]byte(countEnv), &env)
	assert.NoError(t, err)
	assert.Equal(t, int64(12), *env.Count)

	res, err := json.Marshal(env)
	assert.NoError(t, err)
	assert.Equal(t, countEnv, string(res))
}

func TestOKEnvelopeEncodingAndDecoding(t *testing.T) {
	okEnvelopes := []string{
		`["OK","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefaaaaa",false,"error: could not connect to the database"]`,
		`["OK","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefaaaaa",true,""]`,
	}

	for _, raw := range okEnvelopes {
		var env OKEnvelope
		err := json.Unmarshal([]byte(raw), &env)
		assert.NoError(t, err)

		asJSON, err := json.Marshal(env)
		assert.NoError(t, err)
		assert.Equal(t, raw, string(asJSON))
	}
}

func TestClosedEnvelopeEncodingAndDecoding(t *testing.T) {
	closeEnvelopes := []string{
		`["CLOSED","_","error: something went wrong"]`,
		`["CLOSED",":1","auth-required: take a selfie and send it to the CIA"]`,
	}

	for _, raw := range closeEnvelopes {
		var env ClosedEnvelope
		err := json.Unmarshal([]byte(raw), &env)
		assert.NoError(t, err)
		assert.Condition(t, func() (success bool) {
			if env.SubscriptionID != "_" && env.SubscriptionID != ":1" {
				return false
			}
			return true
		})

		res, err := json.Marshal(env)
		assert.NoError(t, err)
		assert.Equal(t, raw, string(res))
	}
}

func TestAuthEnvelopeEncodingAndDecoding(t *testing.T) {
	authEnvelopes := []string{
		`["AUTH","kjsabdlasb aslkd kasndkad \"as.kdnbskadb"]`,
		`["AUTH",{"kind":1,"id":"ae1fc7154296569d87ca4663f6bdf448c217d1590d28c85d158557b8b43b4d69","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","created_at":1683660344,"tags":[],"content":"hello world","sig":"94e10947814b1ebe38af42300ecd90c7642763896c4f69506ae97bfdf54eec3c0c21df96b7d95daa74ff3d414b1d758ee95fc258125deebc31df0c6ba9396a51"}]`,
	}

	for _, raw := range authEnvelopes {
		var env AuthEnvelope
		err := json.Unmarshal([]byte(raw), &env)
		assert.NoError(t, err)

		asJSON, err := json.Marshal(env)
		assert.NoError(t, err)
		assert.Equal(t, raw, string(asJSON))
	}
}

func TestParseMessage(t *testing.T) {
	testCases := []struct {
		Name             string
		Message          []byte
		ExpectedEnvelope Envelope
	}{
		{
			Name:             "nil",
			Message:          nil,
			ExpectedEnvelope: nil,
		},
		{
			Name:             "invalid string",
			Message:          []byte("invalid input"),
			ExpectedEnvelope: nil,
		},
		{
			Name:             "invalid string with a comma",
			Message:          []byte("invalid, input"),
			ExpectedEnvelope: nil,
		},
		{
			Name:             "CLOSED envelope",
			Message:          []byte(`["CLOSED",":1","error: we are broken"]`),
			ExpectedEnvelope: &ClosedEnvelope{SubscriptionID: ":1", Reason: "error: we are broken"},
		},
		{
			Name:             "AUTH envelope",
			Message:          []byte(`["AUTH","bisteka"]`),
			ExpectedEnvelope: &AuthEnvelope{Challenge: ptr("bisteka")},
		},
		{
			Name:             "REQ envelope",
			Message:          []byte(`["REQ","million", {"kinds": [1]}, {"kinds": [30023 ], "#d": ["buteko",    "batuke"]}]`),
			ExpectedEnvelope: &ReqEnvelope{SubscriptionID: "million", Filters: Filters{{Kinds: []int{1}}, {Kinds: []int{30023}, Tags: TagMap{"d": []string{"buteko", "batuke"}}}}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			envelope := ParseMessage(testCase.Message)
			if testCase.ExpectedEnvelope == nil && envelope == nil {
				return
			}

			if testCase.ExpectedEnvelope == nil {
				assert.NotNil(t, envelope, "expected nil but got %v\n", envelope)
			}

			assert.Equal(t, testCase.ExpectedEnvelope.String(), envelope.String())
		})
	}
}

func TestParseMessageSIMD(t *testing.T) {
	testCases := []struct {
		Name                   string
		Message                []byte
		ExpectedEnvelope       Envelope
		ExpectedErrorSubstring string
	}{
		{
			Name:                   "nil",
			Message:                nil,
			ExpectedEnvelope:       nil,
			ExpectedErrorSubstring: "parse failed",
		},
		{
			Name:                   "empty string",
			Message:                []byte(""),
			ExpectedEnvelope:       nil,
			ExpectedErrorSubstring: "parse failed",
		},
		{
			Name:                   "invalid input",
			Message:                []byte("invalid input"),
			ExpectedEnvelope:       nil,
			ExpectedErrorSubstring: "parse failed",
		},
		{
			Name:                   "invalid JSON",
			Message:                []byte("{not valid json}"),
			ExpectedEnvelope:       nil,
			ExpectedErrorSubstring: "parse failed",
		},
		{
			Name:                   "invalid REQ",
			Message:                []byte(`["REQ","zzz", {"authors": [23]}]`),
			ExpectedEnvelope:       nil,
			ExpectedErrorSubstring: "not string, but int",
		},
		{
			Name:             "same invalid REQ from before, but now valid",
			Message:          []byte(`["REQ","zzz", {"kinds": [23]}]`),
			ExpectedEnvelope: &ReqEnvelope{SubscriptionID: "zzz", Filters: Filters{{Kinds: []int{23}}}},
		},
		{
			Name:                   "different invalid REQ",
			Message:                []byte(`["REQ","zzz", {"authors": "string"}]`),
			ExpectedEnvelope:       nil,
			ExpectedErrorSubstring: "next item is not",
		},
		{
			Name:                   "yet another",
			Message:                []byte(`["REQ","zzz", {"unknownfield": "_"}]`),
			ExpectedEnvelope:       nil,
			ExpectedErrorSubstring: "unexpected filter field 'unknownfield'",
		},
		{
			Name: "EVENT envelope with subscription id",
			Message: []byte(
				`["EVENT","_",{"kind":1,"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}]`,
			),
			ExpectedEnvelope: &EventEnvelope{SubscriptionID: ptr("_"), Event: Event{Kind: 1, ID: "dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962", PubKey: "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d", CreatedAt: 1644271588, Tags: Tags{}, Content: "now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?", Sig: "230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}},
		},
		{
			Name: "EVENT envelope without subscription id",
			Message: []byte(
				`["EVENT",{"kind":1,"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}]`,
			),
			ExpectedEnvelope: &EventEnvelope{Event: Event{Kind: 1, ID: "dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962", PubKey: "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d", CreatedAt: 1644271588, Tags: Tags{}, Content: "now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?", Sig: "230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}},
		},
		{
			Name:             "AUTH envelope with challenge",
			Message:          []byte(`["AUTH","challenge-string"]`),
			ExpectedEnvelope: &AuthEnvelope{Challenge: ptr("challenge-string")},
		},
		{
			Name: "AUTH envelope with event",
			Message: []byte(
				`["AUTH",  {"kind":22242,"id":"9b86ca5d2a9b4aa09870e710438c2fd2fcdeca12a18b6f17ab9ebcdbc43f1d4a","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","created_at":1740505646,"tags":[["relay","ws://localhost:7777","2"],["challenge","3027526784722639360"]],"content":"","sig":"eceb827c4bba1de0ab8ee43f3e98df71194f5bdde0af27b5cda38e5c4338b5f63d31961acb5e3c119fd00ecef8b469867d060b697dbaa6ecee1906b483bc307d"}]`,
			),
			ExpectedEnvelope: &AuthEnvelope{Event: Event{Kind: 22242, ID: "9b86ca5d2a9b4aa09870e710438c2fd2fcdeca12a18b6f17ab9ebcdbc43f1d4a", PubKey: "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798", CreatedAt: 1740505646, Tags: Tags{Tag{"relay", "ws://localhost:7777", "2"}, Tag{"challenge", "3027526784722639360"}}, Content: "", Sig: "eceb827c4bba1de0ab8ee43f3e98df71194f5bdde0af27b5cda38e5c4338b5f63d31961acb5e3c119fd00ecef8b469867d060b697dbaa6ecee1906b483bc307d"}},
		},
		{
			Name:             "NOTICE envelope",
			Message:          []byte(`["NOTICE","test notice message"]`),
			ExpectedEnvelope: ptr(NoticeEnvelope("test notice message")),
		},
		{
			Name:             "EOSE envelope",
			Message:          []byte(`["EOSE","subscription123"]`),
			ExpectedEnvelope: ptr(EOSEEnvelope("subscription123")),
		},
		{
			Name:             "CLOSE envelope",
			Message:          []byte(`["CLOSE","subscription123"]`),
			ExpectedEnvelope: ptr(CloseEnvelope("subscription123")),
		},
		{
			Name:             "CLOSED envelope",
			Message:          []byte(`["CLOSED","subscription123","reason: test closed"]`),
			ExpectedEnvelope: &ClosedEnvelope{SubscriptionID: "subscription123", Reason: "reason: test closed"},
		},
		{
			Name:             "OK envelope",
			Message:          []byte(`["OK","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefaaaaa",true,""]`),
			ExpectedEnvelope: &OKEnvelope{EventID: "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefaaaaa", OK: true, Reason: ""},
		},
		{
			Name:             "COUNT envelope with just count",
			Message:          []byte(`["COUNT","sub1",{"count":42}]`),
			ExpectedEnvelope: &CountEnvelope{SubscriptionID: "sub1", Count: ptr(int64(42))},
		},
		{
			Name:             "COUNT envelope with count and hll",
			Message:          []byte(`["COUNT","sub1",{"count":42, "hll": "0100000101000000000000040000000001020000000002000000000200000003000002040000000101020001010000000000000007000004010000000200040000020400000000000102000002000004010000010000000301000102030002000301000300010000070000000001000004000102010000000400010002000000000103000100010001000001040100020001000000000000010000020000000000030100000001000400010000000000000901010100000000040000000b030000010100010000010000010000000003000000000000010003000100020000000000010000010100000100000104000200030001000300000001000101000102"}]`),
			ExpectedEnvelope: &CountEnvelope{SubscriptionID: "sub1", Count: ptr(int64(42)), HyperLogLog: []byte{1, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 1, 2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 2, 4, 0, 0, 0, 1, 1, 2, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 4, 1, 0, 0, 0, 2, 0, 4, 0, 0, 2, 4, 0, 0, 0, 0, 0, 1, 2, 0, 0, 2, 0, 0, 4, 1, 0, 0, 1, 0, 0, 0, 3, 1, 0, 1, 2, 3, 0, 2, 0, 3, 1, 0, 3, 0, 1, 0, 0, 7, 0, 0, 0, 0, 1, 0, 0, 4, 0, 1, 2, 1, 0, 0, 0, 4, 0, 1, 0, 2, 0, 0, 0, 0, 1, 3, 0, 1, 0, 1, 0, 1, 0, 0, 1, 4, 1, 0, 2, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 2, 0, 0, 0, 0, 0, 3, 1, 0, 0, 0, 1, 0, 4, 0, 1, 0, 0, 0, 0, 0, 0, 9, 1, 1, 1, 0, 0, 0, 0, 4, 0, 0, 0, 11, 3, 0, 0, 1, 1, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 1, 0, 3, 0, 1, 0, 2, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0, 1, 4, 0, 2, 0, 3, 0, 1, 0, 3, 0, 0, 0, 1, 0, 1, 1, 0, 1, 2}},
		},
		{
			Name:             "REQ envelope",
			Message:          []byte(`["REQ","sub1",   {"until": 999999, "kinds":[1]}]`),
			ExpectedEnvelope: &ReqEnvelope{SubscriptionID: "sub1", Filters: Filters{{Kinds: []int{1}, Until: ptr(Timestamp(999999))}}},
		},
		{
			Name:             "bigger REQ envelope",
			Message:          []byte(`["REQ","sub1z\\\"zzz",         {"authors":["9b86ca5d2a9b4aa09870e710438c2fd2fcdeca12a18b6f17ab9ebcdbc43f1d4a","8eee10b2ce1162b040fdcfdadb4f888c64aaf87531dab28cc0c09fbdea1b663e","0deadebefb3c1a760f036952abf675076343dd8424efdeaa0f1d9803a359ed46"],"since":1740425099,"limit":2,"#x":["<","as"]}, {"kinds": [2345, 112], "#plic": ["a"], "#ploc": ["blblb", "wuwuw"]}]`),
			ExpectedEnvelope: &ReqEnvelope{SubscriptionID: "sub1z\\\"zzz", Filters: Filters{{Authors: []string{"9b86ca5d2a9b4aa09870e710438c2fd2fcdeca12a18b6f17ab9ebcdbc43f1d4a", "8eee10b2ce1162b040fdcfdadb4f888c64aaf87531dab28cc0c09fbdea1b663e", "0deadebefb3c1a760f036952abf675076343dd8424efdeaa0f1d9803a359ed46"}, Since: ptr(Timestamp(1740425099)), Limit: 2, Tags: TagMap{"x": []string{"<", "as"}}}, {Kinds: []int{2345, 112}, Tags: TagMap{"plic": []string{"a"}, "ploc": []string{"blblb", "wuwuw"}}}}},
		},
	}

	for _, testCase := range testCases {
		smp := SIMDMessageParser{AuxIter: &simdjson.Iter{}}

		t.Run(testCase.Name, func(t *testing.T) {
			envelope, err := smp.ParseMessage(testCase.Message)

			if testCase.ExpectedErrorSubstring == "" {
				require.NoError(t, err)
			} else {
				require.Contains(t, err.Error(), testCase.ExpectedErrorSubstring)
				return
			}

			if testCase.ExpectedEnvelope == nil {
				require.Nil(t, envelope, "expected nil but got %v", envelope)
				return
			}

			require.NotNil(t, envelope, "expected non-nil envelope but got nil")
			require.Equal(t, testCase.ExpectedEnvelope, envelope)
		})
	}
}

func ptr[S any](s S) *S { return &s }
