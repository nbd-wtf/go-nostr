package nip60

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/nbd-wtf/go-nostr"
)

type WalletStash struct {
	sync.Mutex
	wallets map[string]*Wallet

	pendingTokens  map[string][]Token        // tokens not yet assigned to a wallet
	pendingHistory map[string][]HistoryEntry // history entries not yet assigned to a wallet
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
	events <-chan *nostr.Event,
	errors chan<- error,
) *WalletStash {
	wl := &WalletStash{
		wallets:        make(map[string]*Wallet, 1),
		pendingTokens:  make(map[string][]Token),
		pendingHistory: make(map[string][]HistoryEntry),
	}

	go func() {
		for evt := range events {
			wl.Lock()
			switch evt.Kind {
			case 37375:
				wallet := &Wallet{}
				if err := wallet.parse(ctx, kr, evt); err != nil {
					wl.Unlock()
					if errors != nil {
						errors <- fmt.Errorf("event %s failed: %w", evt, err)
					}
					continue
				}

				for _, he := range wl.pendingHistory[wallet.Identifier] {
					wallet.History = append(wallet.History, he)
				}
				for _, token := range wl.pendingTokens[wallet.Identifier] {
					wallet.Tokens = append(wallet.Tokens, token)
				}

				wl.wallets[wallet.Identifier] = wallet

			case 7375: // token
				ref := evt.Tags.GetFirst([]string{"a", ""})
				if ref == nil {
					wl.Unlock()
					if errors != nil {
						errors <- fmt.Errorf("event %s missing 'a' tag", evt)
					}
					continue
				}
				spl := strings.SplitN((*ref)[1], ":", 3)
				if len(spl) < 3 {
					wl.Unlock()
					if errors != nil {
						errors <- fmt.Errorf("event %s invalid 'a' tag", evt)
					}
					continue
				}

				token := Token{}
				if err := token.parse(ctx, kr, evt); err != nil {
					wl.Unlock()
					if errors != nil {
						errors <- fmt.Errorf("event %s failed: %w", evt, err)
					}
					continue
				}

				if wallet, ok := wl.wallets[spl[2]]; ok {
					wallet.Tokens = append(wallet.Tokens, token)
				} else {
					wl.pendingTokens[spl[2]] = append(wl.pendingTokens[spl[2]], token)
				}

			case 7376: // history
				ref := evt.Tags.GetFirst([]string{"a", ""})
				if ref == nil {
					wl.Unlock()
					if errors != nil {
						errors <- fmt.Errorf("event %s missing 'a' tag", evt)
					}
					continue
				}
				spl := strings.SplitN((*ref)[1], ":", 3)
				if len(spl) < 3 {
					wl.Unlock()
					if errors != nil {
						errors <- fmt.Errorf("event %s invalid 'a' tag", evt)
					}
					continue
				}

				he := HistoryEntry{}
				if err := he.parse(ctx, kr, evt); err != nil {
					wl.Unlock()
					if errors != nil {
						errors <- fmt.Errorf("event %s failed: %w", evt, err)
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
