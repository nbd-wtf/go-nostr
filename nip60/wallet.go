package nip60

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/nbd-wtf/go-nostr"
)

type Wallet struct {
	wl *WalletStash

	Identifier  string
	Description string
	Name        string
	PrivateKey  *btcec.PrivateKey
	PublicKey   *btcec.PublicKey
	Relays      []string
	Mints       []string
	Tokens      []Token
	History     []HistoryEntry

	temporaryBalance uint64
	tokensMu         sync.Mutex

	event                 *nostr.Event
	tokensPendingDeletion []string
}

func (w *Wallet) Balance() uint64 {
	var sum uint64
	for _, token := range w.Tokens {
		sum += token.Proofs.Amount()
	}
	return sum
}

func (w *Wallet) DisplayName() string {
	if w.Name != "" {
		return fmt.Sprintf("%s (%s)", w.Name, w.Identifier)
	}
	return w.Identifier
}

func (w *Wallet) toEvent(ctx context.Context, kr nostr.Keyer, evt *nostr.Event) error {
	evt.CreatedAt = nostr.Now()
	evt.Kind = 37375
	evt.Tags = make(nostr.Tags, 0, 7)

	pk, err := kr.GetPublicKey(ctx)
	if err != nil {
		return err
	}

	evt.Content, err = kr.Encrypt(
		ctx,
		fmt.Sprintf(`[["balance","%d","sat"],["privkey","%x"]]`, w.Balance(), w.PrivateKey.Serialize()),
		pk,
	)
	if err != nil {
		return err
	}

	evt.Tags = append(evt.Tags,
		nostr.Tag{"d", w.Identifier},
		nostr.Tag{"unit", "sat"},
	)
	if w.Name != "" {
		evt.Tags = append(evt.Tags, nostr.Tag{"name", w.Name})
	}
	if w.Description != "" {
		evt.Tags = append(evt.Tags, nostr.Tag{"description", w.Description})
	}
	for _, relay := range w.Relays {
		evt.Tags = append(evt.Tags, nostr.Tag{"relay", relay})
	}
	for _, mint := range w.Mints {
		evt.Tags = append(evt.Tags, nostr.Tag{"mint", mint})
	}

	err = kr.SignEvent(ctx, evt)
	if err != nil {
		return err
	}

	return nil
}

func (w *Wallet) parse(ctx context.Context, kr nostr.Keyer, evt *nostr.Event) error {
	w.Tokens = make([]Token, 0, 128)
	w.History = make([]HistoryEntry, 0, 128)
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

	essential := 0
	for _, tag := range tags {
		if len(tag) < 2 {
			continue
		}
		switch tag[0] {
		case "d":
			essential++
			w.Identifier = tag[1]
		case "name":
			w.Name = tag[1]
		case "description":
			w.Description = tag[1]
		case "unit":
			essential++
			if tag[1] != "sat" {
				return fmt.Errorf("only 'sat' wallets are supported")
			}
		case "relay":
			w.Relays = append(w.Relays, tag[1])
		case "mint":
			w.Mints = append(w.Mints, tag[1])
		case "privkey":
			essential++
			skb, err := hex.DecodeString(tag[1])
			if err != nil {
				return fmt.Errorf("failed to parse private key: %w", err)
			}
			w.PrivateKey = secp256k1.PrivKeyFromBytes(skb)
			w.PublicKey = w.PrivateKey.PubKey()
		case "balance":
			if len(tag) < 3 {
				return fmt.Errorf("'balance' tag must have at least 3 items")
			}
			if tag[2] != "sat" {
				return fmt.Errorf("only 'sat' wallets are supported")
			}
			v, err := strconv.ParseUint(tag[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid 'balance' %s: %w", tag[1], err)
			}
			w.temporaryBalance = v
		}
	}

	if essential != 3 {
		return fmt.Errorf("missing essential tags")
	}

	return nil
}
