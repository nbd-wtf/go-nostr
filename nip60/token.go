package nip60

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elnosh/gonuts/cashu"
	"github.com/nbd-wtf/go-nostr"
)

type Token struct {
	Mint    string       `json:"mint"`
	Proofs  cashu.Proofs `json:"proofs"`
	Deleted []string     `json:"del,omitempty"`

	mintedAt nostr.Timestamp
	event    *nostr.Event
}

func (t Token) ID() string {
	if t.event != nil {
		return t.event.ID
	}

	return "<not-published>"
}

func (t Token) toEvent(ctx context.Context, kr nostr.Keyer, evt *nostr.Event) error {
	pk, err := kr.GetPublicKey(ctx)
	if err != nil {
		return err
	}

	evt.CreatedAt = t.mintedAt
	evt.Kind = 7375
	evt.Tags = nostr.Tags{}

	content, _ := json.Marshal(t)
	evt.Content, err = kr.Encrypt(
		ctx,
		string(content),
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

func (t *Token) parse(ctx context.Context, kr nostr.Keyer, evt *nostr.Event) error {
	t.event = evt
	t.mintedAt = evt.CreatedAt

	pk, err := kr.GetPublicKey(ctx)
	if err != nil {
		return err
	}

	content, err := kr.Decrypt(ctx, evt.Content, pk)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(content), t); err != nil {
		return fmt.Errorf("failed to parse token content: %w", err)
	}

	t.Mint, _ = nostr.NormalizeHTTPURL(t.Mint)

	return nil
}
