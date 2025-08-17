package nip60

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/elnosh/gonuts/cashu/nuts/nut13"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip60/client"
)

type Wallet struct {
	sync.Mutex
	tokensMu sync.Mutex
	event    *nostr.Event

	pendingDeletions []string // token events that should be deleted

	kr nostr.Keyer

	// PublishUpdate must be set to a function that publishes event to the user relays
	// (if all arguments are their zero values that means it is a wallet update event).
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

	// properties that come in events
	PrivateKey *btcec.PrivateKey
	PublicKey  *btcec.PublicKey
	Mints      []string
	Tokens     []Token
	History    []HistoryEntry
}

func LoadWallet(
	ctx context.Context,
	kr nostr.Keyer,
	pool *nostr.SimplePool,
	relays []string,
) *Wallet {
	return loadWalletFromPool(ctx, kr, pool, relays, false)
}

func LoadWalletWithHistory(
	ctx context.Context,
	kr nostr.Keyer,
	pool *nostr.SimplePool,
	relays []string,
) *Wallet {
	return loadWalletFromPool(ctx, kr, pool, relays, true)
}

func loadWalletFromPool(
	ctx context.Context,
	kr nostr.Keyer,
	pool *nostr.SimplePool,
	relays []string,
	withHistory bool,
) *Wallet {
	pk, err := kr.GetPublicKey(ctx)
	if err != nil {
		return nil
	}

	kinds := []int{17375, 7375}
	if withHistory {
		kinds = append(kinds, 7376)
	}

	eoseChanE := make(chan struct{})
	events := pool.SubscribeManyNotifyEOSE(
		ctx,
		relays,
		nostr.Filter{Kinds: kinds, Authors: []string{pk}},
		eoseChanE,
	)

	eoseChanD := make(chan struct{})
	deletions := pool.SubscribeManyNotifyEOSE(
		ctx,
		relays,
		nostr.Filter{Kinds: []int{5}, Tags: nostr.TagMap{"k": []string{"7375"}}, Authors: []string{pk}},
		eoseChanD,
	)

	eoseChan := make(chan struct{})
	go func() {
		<-eoseChanD
		<-eoseChanE
		close(eoseChan)
	}()

	return loadWallet(ctx, kr, events, deletions, eoseChan)
}

func loadWallet(
	ctx context.Context,
	kr nostr.Keyer,
	events chan nostr.RelayEvent,
	deletions chan nostr.RelayEvent,
	eoseChan chan struct{},
) *Wallet {
	w := &Wallet{
		pendingDeletions: make([]string, 0, 128),
		kr:               kr,
		Stable:           make(chan struct{}),
		Tokens:           make([]Token, 0, 128),
		History:          make([]HistoryEntry, 0, 128),
	}

	eosed := false
	go func() {
		<-eoseChan
		eosed = true

		// check all pending deletions and delete stuff locally
		for _, id := range w.pendingDeletions {
			w.removeDeletedToken(id)
		}
		w.pendingDeletions = nil

		time.Sleep(100 * time.Millisecond) // race condition hack
		close(w.Stable)
	}()

	go func() {
		for ie := range deletions {
			w.Lock()
			if !eosed {
				for tag := range ie.Event.Tags.FindAll("e") {
					w.pendingDeletions = append(w.pendingDeletions, tag[1])
				}
			} else {
				for tag := range ie.Event.Tags.FindAll("e") {
					w.removeDeletedToken(tag[1])
				}
			}
			w.Unlock()
		}
	}()

	go func() {
		for ie := range events {
			w.Lock()
			switch ie.Event.Kind {
			case 17375:
				if err := w.parse(ctx, kr, ie.Event); err != nil {
					if w.Processed != nil {
						w.Processed(ie.Event, err)
					}
					w.Unlock()
					continue
				}

				// if this metadata is newer than what we had, update
				if w.event == nil || ie.Event.CreatedAt > w.event.CreatedAt {
					w.parse(ctx, kr, ie.Event) // this will either fail or set the new metadata
				}
			case 7375: // token
				token := Token{}
				if err := token.parse(ctx, kr, ie.Event); err != nil {
					if w.Processed != nil {
						w.Processed(ie.Event, err)
					}
					w.Unlock()
					continue
				}

				w.tokensMu.Lock()
				if !slices.ContainsFunc(w.Tokens, func(c Token) bool { return c.event.ID == token.event.ID }) {
					w.Tokens = append(w.Tokens, token)
				}
				w.tokensMu.Unlock()

				// keep track tokens that were deleted by this, if they exist
				if !eosed {
					for _, del := range token.Deleted {
						w.pendingDeletions = append(w.pendingDeletions, del)
					}
				} else {
					for _, del := range token.Deleted {
						w.removeDeletedToken(del)
					}
				}

			case 7376: // history
				he := HistoryEntry{}
				if err := he.parse(ctx, kr, ie.Event); err != nil {
					if w.Processed != nil {
						w.Processed(ie.Event, err)
					}
					w.Unlock()
					continue
				}

				if !slices.ContainsFunc(w.History, func(c HistoryEntry) bool { return c.event.ID == he.event.ID }) {
					w.History = append(w.History, he)
				}
			}

			if w.Processed != nil {
				w.Processed(ie.Event, nil)
			}
			w.Unlock()
		}
	}()

	return w
}

// Close waits for pending operations to end
func (w *Wallet) Close() error {
	w.Lock()
	defer w.Unlock()
	return nil
}

func (w *Wallet) removeDeletedToken(eventId string) {
	for t := len(w.Tokens) - 1; t >= 0; t-- {
		token := w.Tokens[t]
		if token.event != nil && token.event.ID == eventId {
			// swap delete
			w.Tokens[t] = w.Tokens[len(w.Tokens)-1]
			w.Tokens = w.Tokens[0 : len(w.Tokens)-1]
		}
	}
}

func (w *Wallet) Balance() uint64 {
	var sum uint64
	for _, token := range w.Tokens {
		sum += token.Proofs.Amount()
	}
	return sum
}

func (w *Wallet) AddMint(ctx context.Context, urls ...string) error {
	if w.PublishUpdate == nil {
		return fmt.Errorf("can't do write operations: missing PublishUpdate function")
	}

	keysetIdList := []string{}
	for i := range w.Mints {
		keysets, err := client.GetAllKeysets(ctx, w.Mints[i])
		if err != nil {
			return err
		}

		for j := range keysets {
			keysetIdList = append(keysetIdList, keysets[j].Id)
		}
	}

	for _, url := range urls {
		url, err := nostr.NormalizeHTTPURL(url)
		if err != nil {
			return err
		}

		if !slices.Contains(w.Mints, url) {
			keysets, err := client.GetAllKeysets(ctx, url)
			if err != nil {
				return err
			}

			for j := range keysets {
				err = nut13.CheckCollidingKeysets(keysetIdList, []string{keysets[j].Id})
				if err != nil {
					return err
				}
				keysetIdList = append(keysetIdList, keysets[j].Id)
			}
			w.Mints = append(w.Mints, url)
		}
	}

	evt := nostr.Event{}
	if err := w.toEvent(ctx, w.kr, &evt); err != nil {
		return err
	}

	w.Lock()
	w.PublishUpdate(evt, nil, nil, nil, false)
	w.Unlock()

	return nil
}

func (w *Wallet) RemoveMint(ctx context.Context, urls ...string) error {
	if w.PublishUpdate == nil {
		return fmt.Errorf("can't do write operations: missing PublishUpdate function")
	}

	for _, url := range urls {
		url, err := nostr.NormalizeHTTPURL(url)
		if err != nil {
			return err
		}

		if idx := slices.Index(w.Mints, url); idx != -1 {
			w.Mints = slices.Delete(w.Mints, idx, idx+1)
		}
	}

	evt := nostr.Event{}
	if err := w.toEvent(ctx, w.kr, &evt); err != nil {
		return err
	}

	w.Lock()
	w.PublishUpdate(evt, nil, nil, nil, false)
	w.Unlock()

	return nil
}

func (w *Wallet) SetPrivateKey(ctx context.Context, privateKey string) error {
	if w.PublishUpdate == nil {
		return fmt.Errorf("can't do write operations: missing PublishUpdate function")
	}

	skb, err := hex.DecodeString(privateKey)
	if err != nil {
		return err
	}
	if len(skb) != 32 {
		return fmt.Errorf("private key must be 32 bytes, got %d", len(skb))
	}

	w.PrivateKey, w.PublicKey = btcec.PrivKeyFromBytes(skb)

	evt := nostr.Event{}
	if err := w.toEvent(ctx, w.kr, &evt); err != nil {
		return err
	}

	w.Lock()
	w.PublishUpdate(evt, nil, nil, nil, false)
	w.Unlock()

	return nil
}

func (w *Wallet) toEvent(ctx context.Context, kr nostr.Keyer, evt *nostr.Event) error {
	evt.CreatedAt = nostr.Now()
	evt.Kind = 17375
	evt.Tags = nostr.Tags{}

	pk, err := kr.GetPublicKey(ctx)
	if err != nil {
		return err
	}

	encryptedTags := make(nostr.Tags, 0, 1+len(w.Mints))
	if w.PrivateKey != nil {
		encryptedTags = append(encryptedTags, nostr.Tag{"privkey", hex.EncodeToString(w.PrivateKey.Serialize())})
	}

	for _, mint := range w.Mints {
		encryptedTags = append(encryptedTags, nostr.Tag{"mint", mint})
	}
	jtags, _ := json.Marshal(encryptedTags)
	evt.Content, err = kr.Encrypt(
		ctx,
		string(jtags),
		pk,
	)
	if err != nil {
		return err
	}

	err = kr.SignEvent(ctx, evt)
	if err != nil {
		return err
	}

	return nil
}

func (w *Wallet) parse(ctx context.Context, kr nostr.Keyer, evt *nostr.Event) error {
	w.event = evt

	pk, err := kr.GetPublicKey(ctx)
	if err != nil {
		return err
	}
	jsonb, err := kr.Decrypt(ctx, evt.Content, pk)
	if err != nil {
		return err
	}
	var tags nostr.Tags
	if len(jsonb) > 0 {
		tags = make(nostr.Tags, 0, 7)
		if err := json.Unmarshal([]byte(jsonb), &tags); err != nil {
			return err
		}
		tags = append(tags, evt.Tags...)
	}

	var mints []string
	var privateKey *btcec.PrivateKey

	for _, tag := range tags {
		if len(tag) < 2 {
			continue
		}
		switch tag[0] {
		case "mint":
			mints = append(mints, tag[1])
		case "privkey":
			skb, err := hex.DecodeString(tag[1])
			if err != nil {
				return fmt.Errorf("failed to parse private key: %w", err)
			}
			privateKey = secp256k1.PrivKeyFromBytes(skb)
		}
	}

	if privateKey != nil {
		w.PrivateKey = privateKey
		w.PublicKey = w.PrivateKey.PubKey()
	}

	w.Mints = mints

	return nil
}
