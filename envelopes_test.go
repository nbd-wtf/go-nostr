package nostr

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, "kjasbdlasvdluiasvd\"kjasbdksab\\d", env)

	res, err := json.Marshal(env)
	assert.NoError(t, err)
	assert.Equal(t, noticeEnv, string(res))
}

func TestEoseEnvelopeEncodingAndDecoding(t *testing.T) {
	eoseEnv := `["EOSE","kjasbdlasvdluiasvd\"kjasbdksab\\d"]`
	var env EOSEEnvelope
	err := json.Unmarshal([]byte(eoseEnv), &env)
	assert.NoError(t, err)
	assert.Equal(t, "kjasbdlasvdluiasvd\"kjasbdksab\\d", env)

	res, err := json.Marshal(env)
	assert.NoError(t, err)
	assert.Equal(t, eoseEnv, string(res))
}

func TestCountEnvelopeEncodingAndDecoding(t *testing.T) {
	countEnv := `["COUNT","z",{"count":12}]`
	var env CountEnvelope
	err := json.Unmarshal([]byte(countEnv), &env)
	assert.NoError(t, err)
	assert.Equal(t, 12, *env.Count)

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

func ptr[S any](s S) *S { return &s }
