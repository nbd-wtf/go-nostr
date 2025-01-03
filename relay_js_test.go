//go:build js

package nostr

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testRelayURL = func() string {
	url := os.Getenv("TEST_RELAY_URL")
	if url != "" {
		return url
	}
	return "wss://nos.lol"
}()

func TestConnectContext(t *testing.T) {
	// relay client
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	r, err := RelayConnect(ctx, testRelayURL)
	assert.NoError(t, err)

	defer r.Close()
}

func TestConnectContextCanceled(t *testing.T) {
	// relay client
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // make ctx expired
	_, err := RelayConnect(ctx, testRelayURL)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestPublish(t *testing.T) {
	// test note to be sent over websocket
	priv, pub := makeKeyPair(t)
	textNote := Event{
		Kind:      KindTextNote,
		Content:   "hello",
		CreatedAt: Timestamp(1672068534), // random fixed timestamp
		Tags:      Tags{[]string{"foo", "bar"}},
		PubKey:    pub,
	}
	err := textNote.Sign(priv)
	assert.NoError(t, err)

	// connect a client and send the text note
	rl := mustRelayConnect(t, testRelayURL)
	err = rl.Publish(context.Background(), textNote)
	assert.NoError(t, err)
}

func makeKeyPair(t *testing.T) (priv, pub string) {
	t.Helper()

	privkey := GeneratePrivateKey()
	pubkey, err := GetPublicKey(privkey)
	assert.NoError(t, err)

	return privkey, pubkey
}

func mustRelayConnect(t *testing.T, url string) *Relay {
	t.Helper()

	rl, err := RelayConnect(context.Background(), url)
	require.NoError(t, err)

	return rl
}
