package nip47

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNWCURI(t *testing.T) {
	uriParts, err := ParseNWCURI("nostr+walletconnect://739b65aa39cd4318708b5ae5ea85d52b758aa1f5502d32cb033eff9115f95f8d?relay=wss://relay.getalby.com/v1&secret=a5aa9fc79d90271f217c599191ce8479a0404d0c2417f85bc5bee18a89c0cb47")
	require.NoError(t, err)
	assert.Equal(t, "739b65aa39cd4318708b5ae5ea85d52b758aa1f5502d32cb033eff9115f95f8d", uriParts.walletPublicKey)
	assert.Equal(t, "a5aa9fc79d90271f217c599191ce8479a0404d0c2417f85bc5bee18a89c0cb47", uriParts.clientSecretKey)
	assert.Equal(t, []string{"wss://relay.getalby.com/v1"}, uriParts.relays)

	_, err = ParseNWCURI("nostr+walletconnect://739b65aa39cd4318708b5ae5ea85d52b758aa1f5502d32cb033eff9115f95f8d?relay=wss://relay.getalby.com/v1")
	assert.Equal(t, "invalid secret", err.Error())
	_, err = ParseNWCURI("nostr+walletconnect://739b65aa39cd4318708b5ae5ea85d52b758aa1f5502d32cb033eff9115f95f8d?secret=a5aa9fc79d90271f217c599191ce8479a0404d0c2417f85bc5bee18a89c0cb47")
	assert.Equal(t, "no relays", err.Error())
	_, err = ParseNWCURI("nostrwalletconnect://739b65aa39cd4318708b5ae5ea85d52b758aa1f5502d32cb033eff9115f95f8d?relay=wss://relay.getalby.com/v1&secret=a5aa9fc79d90271f217c599191ce8479a0404d0c2417f85bc5bee18a89c0cb47")
	assert.Equal(t, "incorrect scheme", err.Error())
}
