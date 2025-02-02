package nip60

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/keyer"
	"github.com/stretchr/testify/require"
)

var testRelays = []string{
	"wss://relay.damus.io",
	"wss://nos.lol",
	"wss://relay.nostr.band",
}

func TestWalletTransfer(t *testing.T) {
	ctx := context.Background()

	// setup first wallet
	sk1 := os.Getenv("NIP60_SECRET_KEY_1")
	if sk1 == "" {
		t.Skip("NIP60_SECRET_KEY_1 not set")
	}
	kr1, err := keyer.NewPlainKeySigner(sk1)
	if err != nil {
		t.Fatal(err)
	}

	pool := nostr.NewSimplePool(ctx)
	stash1 := LoadStash(ctx, kr1, pool, testRelays)
	if stash1 == nil {
		t.Fatal("failed to load stash 1")
	}
	stash1.PublishUpdate = func(event nostr.Event, deleted, received, change *Token, isHistory bool) {
		pool.PublishMany(ctx, testRelays, event)
	}

	// setup second wallet
	sk2 := os.Getenv("NIP60_SECRET_KEY_2")
	if sk2 == "" {
		t.Skip("NIP60_SECRET_KEY_2 not set")
	}
	kr2, err := keyer.NewPlainKeySigner(sk2)
	if err != nil {
		t.Fatal(err)
	}

	stash2 := LoadStash(ctx, kr2, pool, testRelays)
	if stash2 == nil {
		t.Fatal("failed to load stash 2")
	}
	stash2.PublishUpdate = func(event nostr.Event, deleted, received, change *Token, isHistory bool) {
		pool.PublishMany(ctx, testRelays, event)
	}

	// wait for initial load
	select {
	case <-stash1.Stable:
	case <-time.After(15 * time.Second):
		t.Fatal("timeout waiting for stash 1 to load")
	}

	select {
	case <-stash2.Stable:
	case <-time.After(15 * time.Second):
		t.Fatal("timeout waiting for stash 2 to load")
	}

	// ensure wallets exist and have tokens
	w1 := stash1.EnsureWallet(ctx, "test")
	require.Greater(t, w1.Balance(), uint64(0), "wallet 1 has no balance")

	w2 := stash2.EnsureWallet(ctx, "test")
	initialBalance1 := w1.Balance()
	initialBalance2 := w2.Balance()

	t.Logf("initial balances: w1=%d w2=%d", initialBalance1, initialBalance2)

	// send half of wallet 1's balance to wallet 2
	pk2, err := kr2.GetPublicKey(ctx)
	require.NoError(t, err)

	halfBalance := initialBalance1 / 2
	proofs, mint, err := w1.Send(ctx, halfBalance, WithP2PK(pk2))
	require.NoError(t, err)

	// receive token in wallet 2
	err = w2.Receive(ctx, proofs, mint)
	require.NoError(t, err)

	// verify balances
	require.Equal(t, initialBalance1-halfBalance, w1.Balance(), "wallet 1 balance wrong after send")
	require.Equal(t, initialBalance2+halfBalance, w2.Balance(), "wallet 2 balance wrong after receive")

	// now send it back
	pk1, err := kr1.GetPublicKey(ctx)
	require.NoError(t, err)

	proofs, mint, err = w2.Send(ctx, halfBalance, WithP2PK(pk1))
	require.NoError(t, err)

	// receive token back in wallet 1
	err = w1.Receive(ctx, proofs, mint)
	require.NoError(t, err)

	// verify final balances match initial
	require.Equal(t, initialBalance1, w1.Balance(), "wallet 1 final balance wrong")
	require.Equal(t, initialBalance2, w2.Balance(), "wallet 2 final balance wrong")

	t.Logf("final balances: w1=%d w2=%d", w1.Balance(), w2.Balance())
}
