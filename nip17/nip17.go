package nip17

import (
	"context"
	"fmt"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip59"
)

func GetDMRelays(ctx context.Context, pubkey string, pool *nostr.SimplePool, relaysToQuery []string) []string {
	ie := pool.QuerySingle(ctx, relaysToQuery, nostr.Filter{
		Authors: []string{pubkey},
		Kinds:   []int{nostr.KindDMRelayList},
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

func PublishMessage(
	ctx context.Context,
	content string,
	tags nostr.Tags,
	pool *nostr.SimplePool,
	ourRelays []string,
	theirRelays []string,
	kr nostr.Keyer,
	recipientPubKey string,
	modify func(*nostr.Event),
) error {
	toUs, toThem, err := PrepareMessage(ctx, content, tags, kr, recipientPubKey, modify)
	if err != nil {
		return fmt.Errorf("failed to prepare message: %w", err)
	}

	sendErr := fmt.Errorf("failed to send event to ourselves in any of %v", ourRelays)
	publishOrAuth := func(ctx context.Context, url string, event nostr.Event) {
		r, err := pool.EnsureRelay(url)
		if err != nil {
			return
		}

		err = r.Publish(ctx, event)
		if err != nil && strings.HasPrefix(err.Error(), "auth-required:") {
			authErr := r.Auth(ctx, func(ae *nostr.Event) error { return kr.SignEvent(ctx, ae) })
			if authErr == nil {
				err = r.Publish(ctx, event)
			}
		}

		if err != nil {
			return
		}

		sendErr = nil
	}

	// send to ourselves
	for _, url := range ourRelays {
		publishOrAuth(ctx, url, toUs)
	}

	if sendErr != nil {
		return sendErr
	}

	// send to them
	sendErr = fmt.Errorf("failed to send event to them in any of %v", theirRelays)
	for _, url := range theirRelays {
		publishOrAuth(ctx, url, toThem)
	}

	return sendErr
}

func PrepareMessage(
	ctx context.Context,
	content string,
	tags nostr.Tags,
	kr nostr.Keyer,
	recipientPubKey string,
	modify func(*nostr.Event),
) (toUs nostr.Event, toThem nostr.Event, err error) {
	ourPubkey, err := kr.GetPublicKey(ctx)
	if err != nil {
		return nostr.Event{}, nostr.Event{}, err
	}

	rumor := nostr.Event{
		Kind:      nostr.KindDirectMessage,
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
	kr nostr.Keyer,
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

		for ie := range pool.SubscribeMany(ctx, ourRelays, nostr.Filter{
			Kinds: []int{nostr.KindGiftWrap},
			Tags:  nostr.TagMap{"p": []string{pk}},
			Since: &since,
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
