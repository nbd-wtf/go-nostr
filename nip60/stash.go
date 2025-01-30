package nip60

import (
	"context"
	"fmt"
	"iter"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/nbd-wtf/go-nostr"
)

type WalletStash struct {
	sync.Mutex
	wallets map[string]*Wallet

	pendingHistory   map[string][]HistoryEntry // history entries not yet assigned to a wallet
	pendingTokens    map[string][]Token        // tokens not yet assigned to a wallet
	pendingDeletions []string                  // token events that should be deleted

	kr nostr.Keyer

	// PublishUpdate must be set to a function that publishes event to the user relays
	PublishUpdate func(
		event nostr.Event,
		deleted *Token,
		received *Token,
		change *Token,
		isHistory bool,
	)

	// Processed, if not nil, is called every time a received event is processed
	Processed func(*nostr.Event, error)

	// Stable is closed when we have gotten an EOSE from all relays
	Stable chan struct{}
}

func LoadStash(
	ctx context.Context,
	kr nostr.Keyer,
	pool *nostr.SimplePool,
	relays []string,
) *WalletStash {
	return loadStashFromPool(ctx, kr, pool, relays, false)
}

func LoadStashWithHistory(
	ctx context.Context,
	kr nostr.Keyer,
	pool *nostr.SimplePool,
	relays []string,
) *WalletStash {
	return loadStashFromPool(ctx, kr, pool, relays, true)
}

func loadStashFromPool(
	ctx context.Context,
	kr nostr.Keyer,
	pool *nostr.SimplePool,
	relays []string,
	withHistory bool,
) *WalletStash {
	pk, err := kr.GetPublicKey(ctx)
	if err != nil {
		return nil
	}

	kinds := []int{37375, 7375}
	if withHistory {
		kinds = append(kinds, 7375)
	}

	eoseChan := make(chan struct{})
	events := pool.SubManyNotifyEOSE(
		ctx,
		relays,
		nostr.Filters{
			{Kinds: kinds, Authors: []string{pk}},
			{Kinds: []int{5}, Tags: nostr.TagMap{"k": []string{"7375"}}, Authors: []string{pk}},
		},
		eoseChan,
	)

	return loadStash(ctx, kr, events, eoseChan)
}

func loadStash(
	ctx context.Context,
	kr nostr.Keyer,
	events chan nostr.RelayEvent,
	eoseChan chan struct{},
) *WalletStash {
	wl := &WalletStash{
		wallets:          make(map[string]*Wallet, 1),
		pendingTokens:    make(map[string][]Token),
		pendingHistory:   make(map[string][]HistoryEntry),
		pendingDeletions: make([]string, 0, 128),
		kr:               kr,
		Stable:           make(chan struct{}),
	}

	eosed := false
	go func() {
		<-eoseChan
		eosed = true

		// check all pending deletions and delete stuff locally
		for _, id := range wl.pendingDeletions {
			wl.removeDeletedToken(id)
		}
		wl.pendingDeletions = nil

		time.Sleep(100 * time.Millisecond) // race condition hack
		close(wl.Stable)
	}()

	go func() {
		for ie := range events {
			wl.Lock()
			switch ie.Event.Kind {
			case 5:
				if !eosed {
					for _, tag := range ie.Event.Tags.All([]string{"e", ""}) {
						wl.pendingDeletions = append(wl.pendingDeletions, tag[1])
					}
				} else {
					for _, tag := range ie.Event.Tags.All([]string{"e", ""}) {
						wl.removeDeletedToken(tag[1])
					}
				}
			case 37375:
				wallet := &Wallet{
					wl: wl,
				}
				if err := wallet.parse(ctx, kr, ie.Event); err != nil {
					if wl.Processed != nil {
						wl.Processed(ie.Event, err)
					}
					wl.Unlock()
					continue
				}

				// if we already have a wallet with this identifier then we must be careful
				if curr, ok := wl.wallets[wallet.Identifier]; ok {
					// if the metadata we have is newer ignore this event
					if curr.event != nil && curr.event.CreatedAt > ie.Event.CreatedAt {
						wl.Unlock()
						continue
					}

					// otherwise transfer history events and tokens to the new wallet object
					wallet.Tokens = curr.Tokens
					wallet.History = curr.History
				}

				// get all pending stuff and assign them to this, then delete the pending stuff
				for _, he := range wl.pendingHistory[wallet.Identifier] {
					if !slices.ContainsFunc(wallet.History, func(c HistoryEntry) bool {
						return c.event.ID == he.event.ID
					}) {
						wallet.History = append(wallet.History, he)
					}
				}
				delete(wl.pendingHistory, wallet.Identifier)
				wallet.tokensMu.Lock()
				for _, token := range wl.pendingTokens[wallet.Identifier] {
					if !slices.ContainsFunc(wallet.Tokens, func(c Token) bool {
						return c.event.ID == token.event.ID
					}) {
						wallet.Tokens = append(wallet.Tokens, token)
					}
				}
				delete(wl.pendingTokens, wallet.Identifier)
				wallet.tokensMu.Unlock()

				// finally save the new wallet object
				wl.wallets[wallet.Identifier] = wallet

			case 7375: // token
				ref := ie.Event.Tags.GetFirst([]string{"a", ""})
				if ref == nil {
					if wl.Processed != nil {
						wl.Processed(ie.Event, fmt.Errorf("event missing 'a' tag"))
					}
					wl.Unlock()
					continue
				}
				spl := strings.SplitN((*ref)[1], ":", 3)
				if len(spl) < 3 {
					if wl.Processed != nil {
						wl.Processed(ie.Event, fmt.Errorf("event with invalid 'a' tag"))
					}
					wl.Unlock()
					continue
				}

				token := Token{}
				if err := token.parse(ctx, kr, ie.Event); err != nil {
					if wl.Processed != nil {
						wl.Processed(ie.Event, err)
					}
					wl.Unlock()
					continue
				}

				if wallet, ok := wl.wallets[spl[2]]; ok {
					wallet.tokensMu.Lock()
					if !slices.ContainsFunc(wallet.Tokens, func(c Token) bool {
						return c.event.ID == token.event.ID
					}) {
						wallet.Tokens = append(wallet.Tokens, token)
					}
					wallet.tokensMu.Unlock()
				} else {
					wl.pendingTokens[spl[2]] = append(wl.pendingTokens[spl[2]], token)
				}

				// keep track tokens that were deleted by this, if they exist
				if !eosed {
					for _, del := range token.Deleted {
						wl.pendingDeletions = append(wl.pendingDeletions, del)
					}
				} else {
					for _, del := range token.Deleted {
						wl.removeDeletedToken(del)
					}
				}

			case 7376: // history
				ref := ie.Event.Tags.GetFirst([]string{"a", ""})
				if ref == nil {
					if wl.Processed != nil {
						wl.Processed(ie.Event, fmt.Errorf("event missing 'a' tag"))
					}
					wl.Unlock()
					continue
				}
				spl := strings.SplitN((*ref)[1], ":", 3)
				if len(spl) < 3 {
					if wl.Processed != nil {
						wl.Processed(ie.Event, fmt.Errorf("event with invalid 'a' tag"))
					}
					wl.Unlock()
					continue
				}

				he := HistoryEntry{}
				if err := he.parse(ctx, kr, ie.Event); err != nil {
					if wl.Processed != nil {
						wl.Processed(ie.Event, err)
					}
					wl.Unlock()
					continue
				}

				if wallet, ok := wl.wallets[spl[2]]; ok {
					if !slices.ContainsFunc(wallet.History, func(c HistoryEntry) bool {
						return c.event.ID == he.event.ID
					}) {
						wallet.History = append(wallet.History, he)
					}
				} else {
					wl.pendingHistory[spl[2]] = append(wl.pendingHistory[spl[2]], he)
				}
			}

			if wl.Processed != nil {
				wl.Processed(ie.Event, nil)
			}
			wl.Unlock()
		}
	}()

	return wl
}

func (wl *WalletStash) EnsureWallet(ctx context.Context, id string) *Wallet {
	wl.Lock()
	defer wl.Unlock()
	if w, ok := wl.wallets[id]; ok {
		return w
	}

	sk, err := btcec.NewPrivateKey()
	if err != nil {
		return nil
	}

	w := &Wallet{
		Identifier: id,
		PrivateKey: sk,
		PublicKey:  sk.PubKey(),
		wl:         wl,
	}
	wl.wallets[id] = w

	event := nostr.Event{}
	if err := w.toEvent(ctx, wl.kr, &event); err != nil {
		return nil
	}

	wl.PublishUpdate(event, nil, nil, nil, false)

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

// Close waits for pending operations to end
func (wl *WalletStash) Close() error {
	wl.Lock()
	defer wl.Unlock()
	return nil
}

func (wl *WalletStash) removeDeletedToken(eventId string) {
	for _, w := range wl.wallets {
		for t := len(w.Tokens) - 1; t >= 0; t-- {
			token := w.Tokens[t]
			if token.event != nil && token.event.ID == eventId {
				// swap delete
				w.Tokens[t] = w.Tokens[len(w.Tokens)-1]
				w.Tokens = w.Tokens[0 : len(w.Tokens)-1]
			}
		}
	}
}
