package nip17

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip59"
)

func GetDMRelays(ctx context.Context, pubkey string, pool *nostr.SimplePool, relaysToQuery []string) []string {
	ie := pool.QuerySingle(ctx, relaysToQuery, nostr.Filter{
		Authors: []string{pubkey},
		Kinds:   []int{10050},
	})
	if ie == nil {
		return nil
	}

	res := make([]string, 0, 3)
	for _, tag := range ie.Tags {
		if len(tag) >= 2 && tag[0] == "relay" {
			res = append(res, tag[1])
			if len(res) == 3 {
				return res
			}
		}
	}

	return res
}

func PrepareMessage(
	content string,
	tags nostr.Tags,
	ourPubkey string,
	encrypt func(string) (string, error),
	finalizeAndSign func(*nostr.Event) error,
	recipientPubKey string,
	modify func(*nostr.Event),
) (nostr.Event, error) {
	rumor := nostr.Event{
		Kind:      14,
		Content:   content,
		Tags:      tags,
		CreatedAt: nostr.Now(),
		PubKey:    ourPubkey,
	}
	rumor.ID = rumor.GetID()

	seal, err := nip59.Seal(rumor, encrypt)
	if err != nil {
		return nostr.Event{}, fmt.Errorf("failed to seal: %w", err)
	}

	if err := finalizeAndSign(&seal); err != nil {
		return nostr.Event{}, fmt.Errorf("finalizeAndSign failed: %w", err)
	}

	return nip59.GiftWrap(seal, recipientPubKey, modify)
}

// ListenForMessages returns a channel with the rumors already decrypted and checked
func ListenForMessages(
	ctx context.Context,
	pool *nostr.SimplePool,
	relays []string,
	ourPubkey string,
	since nostr.Timestamp,
	decrypt func(string) (string, error),
) chan nostr.Event {
	ch := make(chan nostr.Event)

	go func() {
		defer close(ch)

		for ie := range pool.SubMany(ctx, relays, nostr.Filters{
			{
				Kinds: []int{1059},
				Tags:  nostr.TagMap{"p": []string{ourPubkey}},
				Since: &since,
			},
		}) {
			seal, err := nip59.GiftUnwrap(*ie.Event, decrypt)
			if err != nil {
				nostr.InfoLogger.Printf("[nip17] failed to unwrap received message: %s\n", err)
				continue
			}

			rumor, err := nip59.Unseal(seal, decrypt)
			if err != nil {
				nostr.InfoLogger.Printf("[nip17] failed to unseal received message: %s\n", err)
				continue
			}

			ch <- rumor
		}
	}()

	return ch
}
