package nip60

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/elnosh/gonuts/cashu"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/keyer"
	"golang.org/x/exp/rand"
)

func TestWallet(t *testing.T) {
	ctx := context.Background()
	kr, err := keyer.NewPlainKeySigner("040cbf11f24b080ad9d8669d7514d9f3b7b1f58e5a6dcb75549352b041656537")
	if err != nil {
		t.Fatal(err)
	}

	privateKey, _ := btcec.NewPrivateKey()

	w := &Wallet{
		kr:         kr,
		PrivateKey: privateKey,
		PublicKey:  privateKey.PubKey(),
		Mints:      []string{"https://mint1.com", "https://mint2.com"},
		Tokens: []Token{
			{
				Mint:     "https://mint1.com",
				Proofs:   cashu.Proofs{{Amount: 100}},
				mintedAt: nostr.Timestamp(time.Now().Add(-3 * time.Hour).Unix()),
			},
			{
				Mint:     "https://mint2.com",
				Proofs:   cashu.Proofs{{Amount: 200}},
				mintedAt: nostr.Timestamp(time.Now().Add(-2 * time.Hour).Unix()),
			},
			{
				Mint:     "https://mint1.com",
				Proofs:   cashu.Proofs{{Amount: 300}},
				mintedAt: nostr.Timestamp(time.Now().Add(-1 * time.Hour).Unix()),
			},
		},
		History: []HistoryEntry{
			{
				In:        true,
				Amount:    100,
				createdAt: nostr.Timestamp(time.Now().Add(-3 * time.Hour).Unix()),
				TokenReferences: []TokenRef{
					{Created: true, EventID: "645babb9051f46ddc97d960e68f82934e627f136dde7b860bf87c9213d937b58"},
				},
			},
			{
				In:        true,
				Amount:    200,
				createdAt: nostr.Timestamp(time.Now().Add(-2 * time.Hour).Unix()),
				TokenReferences: []TokenRef{
					{Created: false, EventID: "add072ae7d7a027748e03024267a1c073f3fbc26cca468ba8630d039a7f5df72"},
					{Created: true, EventID: "b8460b5589b68a0d9a017ac3784d17a0729046206aa631f7f4b763b738e36cf8"},
				},
			},
			{
				In:        true,
				Amount:    300,
				createdAt: nostr.Timestamp(time.Now().Add(-1 * time.Hour).Unix()),
				TokenReferences: []TokenRef{
					{Created: false, EventID: "61f86031d0ab95e9134a3ab955e96104cb1f4d610172838d28aa7ae9dc1cc924"},
					{Created: true, EventID: "588b78e4af06e960434239e7367a0bedf84747d4c52ff943f5e8b7daa3e1b601", IsNutzap: true},
					{Created: false, EventID: "8f14c0a4ff1bf85ccc26bf0125b9a289552f9b59bbb310b163d6a88a7bbd4ebc"},
					{Created: true, EventID: "41a6f442b7c3c9e2f1e8c4835c00f17c56b3e3be4c9f7cf7bc4cdd705b1b61db", IsNutzap: true},
				},
			},
		},
	}

	// turn everything into events
	events := make([]*nostr.Event, 0, 7)

	// wallet metadata event
	metaEvent := &nostr.Event{}
	if err := w.toEvent(ctx, kr, metaEvent); err != nil {
		t.Fatal(err)
	}
	events = append(events, metaEvent)

	// token events
	for i := range w.Tokens {
		evt := &nostr.Event{}
		evt.Tags = nostr.Tags{}
		if err := w.Tokens[i].toEvent(ctx, kr, evt); err != nil {
			t.Fatal(err)
		}
		w.Tokens[i].event = evt
		events = append(events, evt)
	}

	// history events
	for i := range w.History {
		evt := &nostr.Event{}
		evt.Tags = nostr.Tags{}
		if err := w.History[i].toEvent(ctx, kr, evt); err != nil {
			t.Fatal(err)
		}
		w.History[i].event = evt
		events = append(events, evt)
	}

	// test different orderings
	testCases := []struct {
		name string
		sort func([]*nostr.Event)
	}{
		{
			name: "random order",
			sort: func(evts []*nostr.Event) {
				r := rand.New(rand.NewSource(42)) // deterministic
				r.Shuffle(len(evts), func(i, j int) {
					evts[i], evts[j] = evts[j], evts[i]
				})
			},
		},
		{
			name: "most recent first",
			sort: func(evts []*nostr.Event) {
				slices.SortFunc(evts, func(a, b *nostr.Event) int {
					return int(b.CreatedAt - a.CreatedAt)
				})
			},
		},
		{
			name: "least recent first",
			sort: func(evts []*nostr.Event) {
				slices.SortFunc(evts, func(a, b *nostr.Event) int {
					return int(a.CreatedAt - b.CreatedAt)
				})
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// make a copy and sort it
			eventsCopy := make([]*nostr.Event, len(events))
			copy(eventsCopy, events)
			tc.sort(eventsCopy)

			// create relay event channel
			evtChan := make(chan nostr.RelayEvent)
			eoseChan := make(chan struct{})

			// send events in a goroutine
			go func() {
				for _, evt := range eventsCopy {
					evtChan <- nostr.RelayEvent{Event: evt}
				}
				close(eoseChan)
				close(evtChan)
			}()

			// load wallet from events
			loaded := loadWallet(ctx, kr, evtChan, eoseChan)
			loaded.Processed = func(evt *nostr.Event, err error) {
				fmt.Println("processed", evt, err)
			}

			<-loaded.Stable

			// check if loaded wallet matches original
			if len(loaded.Tokens) != len(w.Tokens) {
				t.Errorf("token count mismatch: %d != %d", len(loaded.Tokens), len(w.Tokens))
			}
			if len(loaded.History) != len(w.History) {
				t.Errorf("history count mismatch: %d != %d", len(loaded.History), len(w.History))
			}

			// check tokens are equal regardless of order
			for _, ta := range loaded.Tokens {
				found := false
				for _, tb := range w.Tokens {
					if ta.Mint == tb.Mint && ta.Proofs[0].Amount == tb.Proofs[0].Amount {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("token not found in loaded wallet: %v", ta)
				}
			}

			// check history entries are equal regardless of order
			for _, ha := range loaded.History {
				found := false
				for _, hb := range w.History {
					if ha.In == hb.In && ha.Amount == hb.Amount {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("history entry not found in loaded wallet: %v", ha)
				}
			}
		})
	}
}
