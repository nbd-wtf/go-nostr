package nip60

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/nbd-wtf/go-nostr"
)

type HistoryEntry struct {
	In     bool // in = received, out = sent
	Amount uint64

	tokenEventIDs []string
	nutZaps       []bool

	createdAt nostr.Timestamp
	event     *nostr.Event
}

func (h HistoryEntry) toEvent(ctx context.Context, kr nostr.Keyer, walletId string, evt *nostr.Event) error {
	pk, err := kr.GetPublicKey(ctx)
	if err != nil {
		return err
	}

	dir := "in"
	if !h.In {
		dir = "out"
	}

	evt.CreatedAt = h.createdAt
	evt.Kind = 7376
	evt.Tags = nostr.Tags{{"a", fmt.Sprintf("37375:%s:%s", pk, walletId)}}

	encryptedTags := nostr.Tags{
		nostr.Tag{"direction", dir},
		nostr.Tag{"amount", strconv.FormatUint(uint64(h.Amount), 10), "sat"},
	}

	for i, tid := range h.tokenEventIDs {
		isNutZap := h.nutZaps[i]

		if h.In && isNutZap {
			evt.Tags = append(evt.Tags, nostr.Tag{"e", tid, "", "redeemed"})
			continue
		}

		marker := "created"
		if !h.In {
			marker = "destroyed"
		}

		encryptedTags = append(encryptedTags, nostr.Tag{"e", tid, "", marker})
	}

	jsonb, _ := json.Marshal(encryptedTags)
	evt.Content, err = kr.Encrypt(
		ctx,
		string(jsonb),
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

func (h *HistoryEntry) parse(ctx context.Context, kr nostr.Keyer, evt *nostr.Event) error {
	h.event = evt
	h.createdAt = evt.CreatedAt

	pk, err := kr.GetPublicKey(ctx)
	if err != nil {
		return err
	}

	// event tags and encrypted tags are mixed together
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
		case "direction":
			essential++
			if tag[1] == "in" {
				h.In = true
			} else if tag[1] == "out" {
				h.In = false
			} else {
				return fmt.Errorf("unexpected 'direction' tag %s", tag[1])
			}
		case "amount":
			essential++
			if len(tag) < 3 {
				return fmt.Errorf("'amount' tag must have at least 3 items")
			}
			if tag[2] != "sat" {
				return fmt.Errorf("only 'sat' wallets are supported")
			}
			v, err := strconv.ParseUint(tag[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid 'amount' %s: %w", tag[1], err)
			}
			h.Amount = v
		case "e":
			essential++
			if len(tag) < 4 {
				return fmt.Errorf("'e' tag must have at least 4 items")
			}
			if !nostr.IsValid32ByteHex(tag[1]) {
				return fmt.Errorf("'e' tag has invalid event id %s", tag[1])
			}
			h.tokenEventIDs = append(h.tokenEventIDs, tag[1])
			switch tag[3] {
			case "created":
				h.nutZaps = append(h.nutZaps, false)
			case "destroyed":
				h.nutZaps = append(h.nutZaps, false)
			case "redeemed":
				h.nutZaps = append(h.nutZaps, true)
			}
		}
	}

	if essential < 3 {
		return fmt.Errorf("missing essential tags")
	}

	return nil
}
