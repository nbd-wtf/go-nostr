package nip77

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip77/negentropy"
)

func NegentropySync(ctx context.Context, store nostr.RelayStore, url string, filter nostr.Filter) error {
	id := "go-nostr-tmp" // for now we can't have more than one subscription in the same connection

	data, err := store.QuerySync(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to query our local store: %w", err)
	}

	neg := negentropy.NewNegentropy(negentropy.NewVector(), 1024*1024)
	for _, evt := range data {
		neg.Insert(evt)
	}

	result := make(chan error)

	var r *nostr.Relay
	r, err = nostr.RelayConnect(ctx, url, nostr.WithCustomHandler(func(data []byte) {
		envelope := ParseNegMessage(data)
		if envelope == nil {
			return
		}
		switch env := envelope.(type) {
		case *OpenEnvelope, *CloseEnvelope:
			result <- fmt.Errorf("unexpected %s received from relay", env.Label())
			return
		case *ErrorEnvelope:
			result <- fmt.Errorf("relay returned a %s: %s", env.Label(), env.Reason)
			return
		case *MessageEnvelope:
			msg, err := hex.DecodeString(env.Message)
			if err != nil {
				result <- fmt.Errorf("relay sent invalid message: %w", err)
				return
			}

			nextmsg, err := neg.Reconcile(msg)
			if err != nil {
				result <- fmt.Errorf("failed to reconcile: %w", err)
				return
			}

			msgb, _ := MessageEnvelope{id, hex.EncodeToString(nextmsg)}.MarshalJSON()
			r.Write(msgb)
		}
	}))
	if err != nil {
		return err
	}

	msg := neg.Initiate()
	open, _ := OpenEnvelope{id, filter, hex.EncodeToString(msg)}.MarshalJSON()
	err = <-r.Write(open)
	if err != nil {
		return fmt.Errorf("failed to write to relay: %w", err)
	}

	defer func() {
		clse, _ := CloseEnvelope{id}.MarshalJSON()
		r.Write(clse)
	}()

	for _, p := range []struct {
		items  chan string
		source nostr.RelayStore
		target nostr.RelayStore
	}{{neg.Haves, store, r}, {neg.HaveNots, r, store}} {
		p := p
		go func() {
			for item := range p.items {
				evts, _ := p.source.QuerySync(ctx, nostr.Filter{IDs: []string{item}})
				for _, evt := range evts {
					p.target.Publish(ctx, *evt)
				}
			}
		}()
	}

	err = <-result
	if err != nil {
		return err
	}

	return nil
}
