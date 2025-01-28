package nip60

import (
	"context"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/elnosh/gonuts/cashu"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/keyer"
	"github.com/stretchr/testify/require"
)

func TestWalletRoundtrip(t *testing.T) {
	ctx := context.Background()
	kr, err := keyer.NewPlainKeySigner("94b46586f475bbc92746cb8f14d59b083047ac3ab747774b066d17673c1cc527")
	require.NoError(t, err)

	// create initial wallets with arbitrary data
	sk1, _ := btcec.NewPrivateKey()
	wallet1 := Wallet{
		Identifier:  "wallet1",
		Name:        "My First Wallet",
		Description: "Test wallet number one",
		PrivateKey:  sk1,
		Relays:      []string{"wss://relay1.example.com", "wss://relay2.example.com"},
		Mints:       []string{"https://mint1.example.com"},
		Tokens: []Token{
			{
				Mint: "https://mint1.example.com",
				Proofs: []cashu.Proof{
					{Id: "proof1", Amount: 100, Secret: "secret1", C: "c1"},
					{Id: "proof2", Amount: 200, Secret: "secret2", C: "c2"},
				},
				mintedAt: nostr.Now(),
			},
			{
				Mint: "https://mint2.example.com",
				Proofs: []cashu.Proof{
					{Id: "proof3", Amount: 500, Secret: "secret3", C: "c3"},
				},
				mintedAt: nostr.Now(),
			},
		},
		History: []HistoryEntry{
			{
				In:     true,
				Amount: 300,
				tokenEventIDs: []string{
					"559cecf5aba6ab825347bedfd56ff603a2c6aa7c8d88790ca1e232759699bbc7",
					"8f2c40b064e3e601d070362f53ace6fe124992da8a7322357c0868f22f6c2350",
				},
				nutZaps:   []bool{false, false},
				createdAt: nostr.Now(),
			},
		},
	}

	sk2, _ := btcec.NewPrivateKey()
	wallet2 := Wallet{
		Identifier:  "wallet2",
		Name:        "Second Wallet",
		Description: "Test wallet number two",
		PrivateKey:  sk2,
		Relays:      []string{"wss://relay3.example.com"},
		Mints:       []string{"https://mint2.example.com"},
		Tokens: []Token{
			{
				Mint: "https://mint2.example.com",
				Proofs: []cashu.Proof{
					{Id: "proof3", Amount: 500, Secret: "secret3", C: "c3"},
				},
				mintedAt: nostr.Now(),
			},
		},
		History: []HistoryEntry{
			{
				In:     false,
				Amount: 200,
				tokenEventIDs: []string{
					"cc9dd6298ae7e1ae0866448f11fed1c3a818b7db837caf8d5c48e496200477fe",
				},
				nutZaps:   []bool{false},
				createdAt: nostr.Now(),
			},
			{
				In:     true,
				Amount: 300,
				tokenEventIDs: []string{
					"63e8ff4ca4f16d6edc0c93dd1659cc8029178560aef2c9a00ca323738ed680e3",
					"3898e1c01fd6043dd46b819ce6a940867ccc116bc7c733124d2c0658fb1d569e",
				},
				nutZaps:   []bool{false, false},
				createdAt: nostr.Now(),
			},
		},
	}

	// convert wallets to events
	events := [][]nostr.Event{
		make([]nostr.Event, 0, 4),
		make([]nostr.Event, 0, 4),
	}

	for i, w := range []*Wallet{&wallet1, &wallet2} {
		evt := nostr.Event{}
		err := w.toEvent(ctx, kr, &evt)
		require.NoError(t, err)
		events[i] = append(events[i], evt)

		for _, token := range w.Tokens {
			evt = nostr.Event{}
			err = token.toEvent(ctx, kr, w.Identifier, &evt)
			require.NoError(t, err)
			events[i] = append(events[i], evt)
		}

		for _, he := range w.History {
			evt = nostr.Event{}
			err = he.toEvent(ctx, kr, w.Identifier, &evt)
			require.NoError(t, err)
			events[i] = append(events[i], evt)
		}
	}

	events1, events2 := events[0], events[1]

	// combine all events
	allEvents := append(events1, events2...)
	require.Len(t, allEvents, 8)

	// make a derived shuffled version
	reversedAllEvents := make([]nostr.Event, len(allEvents))
	for i, evt := range allEvents {
		reversedAllEvents[len(allEvents)-1-i] = evt
	}

	for _, allEvents := range [][]nostr.Event{allEvents, reversedAllEvents} {
		// create channel and feed events into it
		eventChan := make(chan nostr.RelayEvent)
		done := make(chan struct{})
		go func() {
			for _, evt := range allEvents {
				eventChan <- nostr.RelayEvent{Event: &evt}
			}
			close(eventChan)
			done <- struct{}{}
		}()

		// load wallets from events
		walletStash := loadStash(ctx, kr, eventChan, make(chan struct{}))

		var errorChanErr error
		go func() {
			for {
				errorChanErr = <-walletStash.Processed
				if errorChanErr != nil {
					return
				}
			}
		}()

		<-done
		time.Sleep(time.Millisecond * 200)
		require.NoError(t, errorChanErr, "errorChan shouldn't have received any errors: %w", errorChanErr)

		// compare loaded wallets with original ones
		loadedWallet1 := walletStash.wallets[wallet1.Identifier]
		require.Equal(t, wallet1.Name, loadedWallet1.Name)
		require.Equal(t, wallet1.Description, loadedWallet1.Description)
		require.Equal(t, wallet1.Mints, loadedWallet1.Mints)
		require.Equal(t, wallet1.PrivateKey, loadedWallet1.PrivateKey)
		require.Len(t, loadedWallet1.Tokens, len(wallet1.Tokens))
		require.Len(t, loadedWallet1.History, len(wallet1.History))

		loadedWallet2 := walletStash.wallets[wallet2.Identifier]
		require.Equal(t, wallet2.Name, loadedWallet2.Name)
		require.Equal(t, wallet2.Description, loadedWallet2.Description)
		require.Equal(t, wallet2.Mints, loadedWallet2.Mints)
		require.Equal(t, wallet2.PrivateKey, loadedWallet2.PrivateKey)
		require.Len(t, loadedWallet2.Tokens, len(wallet2.Tokens))
		require.Len(t, loadedWallet2.History, len(wallet2.History))

		// check token amounts
		require.Equal(t, wallet1.Balance(), loadedWallet1.Balance())
		require.Equal(t, wallet2.Balance(), loadedWallet2.Balance())
	}
}
