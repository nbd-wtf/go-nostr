package libsecp256k1

import (
	"encoding/json"
	"testing"

	"github.com/nbd-wtf/go-nostr/core"
	"github.com/nbd-wtf/go-nostr/test_common"
	"github.com/stretchr/testify/assert"
)

func TestEventVerification(t *testing.T) {
	for _, jevt := range test_common.NormalEvents {
		evt := core.Event{}
		json.Unmarshal([]byte(jevt), &evt)
		ok, _ := CheckSignature(evt)
		shouldBe, _ := evt.CheckSignature()
		assert.Equal(t, ok, shouldBe, "%s signature must be %s", jevt, shouldBe)
	}
}
