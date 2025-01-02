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

func TestConnectContext(t *testing.T) {
	url := os.Getenv("TEST_RELAY_URL")
	if url == "" {
		t.Fatal("please set the environment: $TEST_RELAY_URL")
	}

	// relay client
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	r, err := RelayConnect(ctx, url)
	assert.NoError(t, err)

	defer r.Close()
}

func TestConnectContextCanceled(t *testing.T) {
	url := os.Getenv("TEST_RELAY_URL")
	if url == "" {
		t.Fatal("please set the environment: $TEST_RELAY_URL")
	}

	// relay client
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // make ctx expired
	_, err := RelayConnect(ctx, url)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestPublish(t *testing.T) {
	url := os.Getenv("TEST_RELAY_URL")
	if url == "" {
		t.Fatal("please set the environment: $TEST_RELAY_URL")
	}

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
	rl := mustRelayConnect(t, url)
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
