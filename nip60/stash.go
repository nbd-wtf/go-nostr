package nip60

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"sync"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/nbd-wtf/go-nostr"
)

type WalletStash struct {
	sync.Mutex
	wallets map[string]*Wallet

	pendingTokens  map[string][]Token        // tokens not yet assigned to a wallet
	pendingHistory map[string][]HistoryEntry // history entries not yet assigned to a wallet
}

func (wl *WalletStash) EnsureWallet(id string) *Wallet {
	wl.Lock()
	defer wl.Unlock()
	if w, ok := wl.wallets[id]; ok {
		return w
	}

	sk, err := btcec.NewPrivateKey()
	if err != nil {
		panic(err)
	}

	w := &Wallet{
		Identifier: id,
		PrivateKey: sk,
		PublicKey:  sk.PubKey(),
	}
	wl.wallets[id] = w
	return w
}

func (wl *WalletStash) Wallets() iter.Seq[*Wallet] {
	return func(yield func(*Wallet) bool) {
		wl.Lock()
		defer wl.Unlock()

		for _, w := range wl.wallets {
			if !yield(w) {
				return
			}
		}
	}
}

func NewStash() *WalletStash {
	return &WalletStash{
		wallets:        make(map[string]*Wallet, 1),
		pendingTokens:  make(map[string][]Token),
		pendingHistory: make(map[string][]HistoryEntry),
	}
}

func LoadStash(
	ctx context.Context,
	kr nostr.Keyer,
	events <-chan nostr.RelayEvent,
	errors chan<- error,
) *WalletStash {
	wl := &WalletStash{
		wallets:        make(map[string]*Wallet, 1),
		pendingTokens:  make(map[string][]Token),
		pendingHistory: make(map[string][]HistoryEntry),
	}

	go func() {
		for ie := range events {
			wl.Lock()
			switch ie.Event.Kind {
			case 37375:
				wallet := &Wallet{}
				if err := wallet.parse(ctx, kr, ie.Event); err != nil {
					wl.Unlock()
					if errors != nil {
						errors <- fmt.Errorf("event %s failed: %w", ie.Event, err)
					}
					continue
				}

				for _, he := range wl.pendingHistory[wallet.Identifier] {
					wallet.History = append(wallet.History, he)
				}

				wallet.tokensMu.Lock()
				for _, token := range wl.pendingTokens[wallet.Identifier] {
					wallet.Tokens = append(wallet.Tokens, token)
				}
				wallet.tokensMu.Unlock()

				wl.wallets[wallet.Identifier] = wallet

			case 7375: // token
				ref := ie.Event.Tags.GetFirst([]string{"a", ""})
				if ref == nil {
					wl.Unlock()
					if errors != nil {
						errors <- fmt.Errorf("event %s missing 'a' tag", ie.Event)
					}
					continue
				}
				spl := strings.SplitN((*ref)[1], ":", 3)
				if len(spl) < 3 {
					wl.Unlock()
					if errors != nil {
						errors <- fmt.Errorf("event %s invalid 'a' tag", ie.Event)
					}
					continue
				}

				token := Token{}
				if err := token.parse(ctx, kr, ie.Event); err != nil {
					wl.Unlock()
					if errors != nil {
						errors <- fmt.Errorf("event %s failed: %w", ie.Event, err)
					}
					continue
				}

				if wallet, ok := wl.wallets[spl[2]]; ok {
					wallet.tokensMu.Lock()
					wallet.Tokens = append(wallet.Tokens, token)
					wallet.tokensMu.Unlock()
				} else {
					wl.pendingTokens[spl[2]] = append(wl.pendingTokens[spl[2]], token)
				}

			case 7376: // history
				ref := ie.Event.Tags.GetFirst([]string{"a", ""})
				if ref == nil {
					wl.Unlock()
					if errors != nil {
						errors <- fmt.Errorf("event %s missing 'a' tag", ie.Event)
					}
					continue
				}
				spl := strings.SplitN((*ref)[1], ":", 3)
				if len(spl) < 3 {
					wl.Unlock()
					if errors != nil {
						errors <- fmt.Errorf("event %s invalid 'a' tag", ie.Event)
					}
					continue
				}

				he := HistoryEntry{}
				if err := he.parse(ctx, kr, ie.Event); err != nil {
					wl.Unlock()
					if errors != nil {
						errors <- fmt.Errorf("event %s failed: %w", ie.Event, err)
					}
					continue
				}

				if wallet, ok := wl.wallets[spl[2]]; ok {
					wallet.History = append(wallet.History, he)
				} else {
					wl.pendingHistory[spl[2]] = append(wl.pendingHistory[spl[2]], he)
				}
			}

			wl.Unlock()
		}
	}()

	return wl
}
