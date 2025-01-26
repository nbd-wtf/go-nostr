package nip60

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/nbd-wtf/go-nostr"
)

type Wallet struct {
	Identifier  string
	Description string
	Name        string
	PrivateKey  string
	Relays      []string
	Mints       []string
	Tokens      []Token
	History     []HistoryEntry

	temporaryBalance uint32
}

func (w Wallet) Balance() uint32 {
	var sum uint32
	for _, token := range w.Tokens {
		sum += token.Amount()
	}
	return sum
}

func (w Wallet) ToPublishableEvents(ctx context.Context, kr nostr.Keyer, skipExisting bool) ([]nostr.Event, error) {
	evt := nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      37375,
		Tags:      make(nostr.Tags, 0, 7),
	}

	pk, err := kr.GetPublicKey(ctx)
	if err != nil {
		return nil, err
	}

	evt.Content, err = kr.Encrypt(
		ctx,
		fmt.Sprintf(`[["balance","%d","sat"],["privkey","%s"]]`, w.Balance(), w.PrivateKey),
		pk,
	)
	if err != nil {
		return nil, err
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

	err = kr.SignEvent(ctx, &evt)
	if err != nil {
		return nil, err
	}

	events := make([]nostr.Event, 0, 1+len(w.Tokens))
	events = append(events, evt)

	for _, t := range w.Tokens {
		var evt nostr.Event

		if t.event != nil {
			if skipExisting {
				continue
			}
			evt = *t.event
		} else {
			err := t.toEvent(ctx, kr, w.Identifier, &evt)
			if err != nil {
				return nil, err
			}
		}

		events = append(events, evt)
	}

	for _, h := range w.History {
		var evt nostr.Event

		if h.event != nil {
			if skipExisting {
				continue
			}
			evt = *h.event
		} else {
			err := h.toEvent(ctx, kr, w.Identifier, &evt)
			if err != nil {
				return nil, err
			}
		}

		events = append(events, evt)
	}

	return events, nil
}

func (w *Wallet) parse(ctx context.Context, kr nostr.Keyer, evt *nostr.Event) error {
	w.Tokens = make([]Token, 0, 128)
	w.History = make([]HistoryEntry, 0, 128)

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
			w.PrivateKey = tag[1]
		case "balance":
			if len(tag) < 3 {
				return fmt.Errorf("'balance' tag must have at least 3 items")
			}
			if tag[2] != "sat" {
				return fmt.Errorf("only 'sat' wallets are supported")
			}
			v, err := strconv.ParseUint(tag[1], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid 'balance' %s: %w", tag[1], err)
			}
			w.temporaryBalance = uint32(v)
		}
	}

	return nil
}
