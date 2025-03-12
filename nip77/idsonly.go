package nip77

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip77/negentropy"
	"github.com/nbd-wtf/go-nostr/nip77/negentropy/storage/empty"
)

func FetchIDsOnly(
	ctx context.Context,
	url string,
	filter nostr.Filter,
) (<-chan string, error) {
	id := "go-nostr-tmp" // for now we can't have more than one subscription in the same connection

	neg := negentropy.New(empty.Empty{}, 1024*1024)
	result := make(chan error)

	var r *nostr.Relay
	r, err := nostr.RelayConnect(ctx, url, nostr.WithCustomHandler(func(data string) {
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
		return nil, err
	}

	msg := neg.Start()
	open, _ := OpenEnvelope{id, filter, msg}.MarshalJSON()
	err = <-r.Write(open)
	if err != nil {
		return nil, fmt.Errorf("failed to write to relay: %w", err)
	}

	ch := make(chan string)
	go func() {
		for id := range neg.HaveNots {
			ch <- id
		}
		clse, _ := CloseEnvelope{id}.MarshalJSON()
		r.Write(clse)
		close(ch)
	}()

	return ch, nil
}
