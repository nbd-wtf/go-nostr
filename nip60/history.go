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

	TokenReferences []TokenRef

	createdAt nostr.Timestamp
	event     *nostr.Event
}

type TokenRef struct {
	EventID  string
	Created  bool
	IsNutzap bool
}

func (h HistoryEntry) toEvent(ctx context.Context, kr nostr.Keyer, evt *nostr.Event) error {
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
	evt.Tags = nostr.Tags{}

	encryptedTags := nostr.Tags{
		nostr.Tag{"direction", dir},
		nostr.Tag{"amount", strconv.FormatUint(uint64(h.Amount), 10)},
	}

	for _, tf := range h.TokenReferences {
		if tf.IsNutzap {
			evt.Tags = append(evt.Tags, nostr.Tag{"e", tf.EventID, "", "redeemed"})
			continue
		}

		marker := "destroyed"
		if tf.Created {
			marker = "created"
		}

		encryptedTags = append(encryptedTags, nostr.Tag{"e", tf.EventID, "", marker})
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
	h.TokenReferences = make([]TokenRef, 0, 3)

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

	missingDirection := true
	for _, tag := range tags {
		if len(tag) < 2 {
			continue
		}
		switch tag[0] {
		case "direction":
			missingDirection = false
			if tag[1] == "in" {
				h.In = true
			} else if tag[1] == "out" {
				h.In = false
			} else {
				return fmt.Errorf("unexpected 'direction' tag %s", tag[1])
			}
		case "amount":
			if len(tag) < 2 {
				return fmt.Errorf("'amount' tag must have at least 2 items")
			}
			v, err := strconv.ParseUint(tag[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid 'amount' %s: %w", tag[1], err)
			}
			h.Amount = v
		case "e":
			if len(tag) < 4 {
				return fmt.Errorf("'e' tag must have at least 4 items")
			}
			if !nostr.IsValid32ByteHex(tag[1]) {
				return fmt.Errorf("'e' tag has invalid event id %s", tag[1])
			}

			tf := TokenRef{EventID: tag[1]}
			switch tag[3] {
			case "created":
				tf.Created = true
			case "destroyed":
				tf.Created = false
			case "redeemed":
				tf.IsNutzap = true
				tf.Created = true
			default:
				return fmt.Errorf("unsupported 'e' token marker: %s", tag[3])
			}
			h.TokenReferences = append(h.TokenReferences, tf)
		}
	}

	if h.Amount == 0 {
		return fmt.Errorf("missing 'amount' tag")
	}

	if missingDirection {
		return fmt.Errorf("missing 'direction' tag")
	}

	return nil
}
