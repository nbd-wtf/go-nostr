package nip60

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

type Token struct {
	Mint   string  `json:"mint"`
	Proofs []Proof `json:"proofs"`

	mintedAt nostr.Timestamp
	event    *nostr.Event
}

type Proof struct {
	ID     string `json:"id"`
	Amount uint32 `json:"amount"`
	Secret string `json:"secret"`
	C      string `json:"C"`
}

func (t Token) Amount() uint32 {
	var sum uint32
	for _, p := range t.Proofs {
		sum += p.Amount
	}
	return sum
}

func (t Token) toEvent(ctx context.Context, kr nostr.Keyer, walletId string, evt *nostr.Event) error {
	pk, err := kr.GetPublicKey(ctx)
	if err != nil {
		return err
	}

	evt.CreatedAt = t.mintedAt
	evt.Kind = 7375
	evt.Tags = nostr.Tags{{"a", fmt.Sprintf("37375:%s:%s", pk, walletId)}}

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

	return nil
}
