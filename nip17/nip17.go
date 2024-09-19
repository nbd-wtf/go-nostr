package nip17

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/keyer"
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
	ctx context.Context,
	content string,
	tags nostr.Tags,
	kr keyer.Keyer,
	recipientPubKey string,
	modify func(*nostr.Event),
) (toUs nostr.Event, toThem nostr.Event, err error) {
	ourPubkey, err := kr.GetPublicKey(ctx)
	if err != nil {
		return nostr.Event{}, nostr.Event{}, err
	}

	rumor := nostr.Event{
		Kind:      14,
		Content:   content,
		Tags:      append(tags, nostr.Tag{"p", recipientPubKey}),
		CreatedAt: nostr.Now(),
		PubKey:    ourPubkey,
	}
	rumor.ID = rumor.GetID()

	toUs, err = nip59.GiftWrap(
		rumor,
		ourPubkey,
		func(s string) (string, error) { return kr.Encrypt(ctx, s, ourPubkey) },
		func(e *nostr.Event) error { return kr.SignEvent(ctx, e) },
		modify,
	)
	if err != nil {
		return nostr.Event{}, nostr.Event{}, err
	}

	toThem, err = nip59.GiftWrap(
		rumor,
		recipientPubKey,
		func(s string) (string, error) { return kr.Encrypt(ctx, s, recipientPubKey) },
		func(e *nostr.Event) error { return kr.SignEvent(ctx, e) },
		modify,
	)
	if err != nil {
		return nostr.Event{}, nostr.Event{}, err
	}

	return toUs, toThem, nil
}

// ListenForMessages returns a channel with the rumors already decrypted and checked
func ListenForMessages(
	ctx context.Context,
	pool *nostr.SimplePool,
	kr keyer.Keyer,
	ourRelays []string,
	since nostr.Timestamp,
) chan nostr.Event {
	ch := make(chan nostr.Event)

	go func() {
		defer close(ch)

		pk, err := kr.GetPublicKey(ctx)
		if err != nil {
			nostr.InfoLogger.Printf("[nip17] failed to get public key from Keyer: %s\n", err)
			return
		}

		for ie := range pool.SubMany(ctx, ourRelays, nostr.Filters{
			{
				Kinds: []int{1059},
				Tags:  nostr.TagMap{"p": []string{pk}},
				Since: &since,
			},
		}) {
			rumor, err := nip59.GiftUnwrap(
				*ie.Event,
				func(otherpubkey, ciphertext string) (string, error) { return kr.Decrypt(ctx, ciphertext, otherpubkey) },
			)
			if err != nil {
				nostr.InfoLogger.Printf("[nip17] failed to unwrap received message '%s' from %s: %s\n", ie.Event, ie.Relay.URL, err)
				continue
			}

			ch <- rumor
		}
	}()

	return ch
}
