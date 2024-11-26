package nip77

import (
	"context"
	"fmt"
	"sync"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip77/negentropy"
	"github.com/nbd-wtf/go-nostr/nip77/negentropy/storage/vector"
)

func NegentropySync(ctx context.Context, store nostr.RelayStore, url string, filter nostr.Filter) error {
	id := "go-nostr-tmp" // for now we can't have more than one subscription in the same connection

	data, err := store.QuerySync(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to query our local store: %w", err)
	}

	vec := vector.New()
	neg := negentropy.New(vec, 1024*1024)
	for _, evt := range data {
		vec.Insert(evt.CreatedAt, evt.ID)
	}
	vec.Seal()

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
			nextmsg, err := neg.Reconcile(env.Message)
			if err != nil {
				result <- fmt.Errorf("failed to reconcile: %w", err)
				return
			}

			if nextmsg != "" {
				msgb, _ := MessageEnvelope{id, nextmsg}.MarshalJSON()
				r.Write(msgb)
			}
		}
	}))
	if err != nil {
		return err
	}

	msg := neg.Start()
	open, _ := OpenEnvelope{id, filter, msg}.MarshalJSON()
	err = <-r.Write(open)
	if err != nil {
		return fmt.Errorf("failed to write to relay: %w", err)
	}

	defer func() {
		clse, _ := CloseEnvelope{id}.MarshalJSON()
		r.Write(clse)
	}()

	type direction struct {
		label  string
		items  chan string
		source nostr.RelayStore
		target nostr.RelayStore
	}

	wg := sync.WaitGroup{}
	pool := newidlistpool(50)
	for _, dir := range []direction{
		{"up", neg.Haves, store, r},
		{"down", neg.HaveNots, r, store},
	} {
		wg.Add(1)
		go func(dir direction) {
			defer wg.Done()

			seen := make(map[string]struct{})

			doSync := func(ids []string) {
				defer wg.Done()
				defer pool.giveback(ids)

				if len(ids) == 0 {
					return
				}
				evtch, err := dir.source.QueryEvents(ctx, nostr.Filter{IDs: ids})
				if err != nil {
					result <- fmt.Errorf("error querying source on %s: %w", dir.label, err)
					return
				}
				for evt := range evtch {
					dir.target.Publish(ctx, *evt)
				}
			}

			ids := pool.grab()
			for item := range dir.items {
				if _, ok := seen[item]; ok {
					continue
				}
				seen[item] = struct{}{}

				ids = append(ids, item)
				if len(ids) == 50 {
					wg.Add(1)
					go doSync(ids)
					ids = pool.grab()
				}
			}
			wg.Add(1)
			doSync(ids)
		}(dir)
	}

	go func() {
		wg.Wait()
		result <- nil
	}()

	err = <-result
	if err != nil {
		return err
	}

	return nil
}
