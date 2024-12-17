package main

import (
	"context"
	"fmt"
	"time"

	"github.com/fiatjaf/eventstore"
	"github.com/fiatjaf/eventstore/slicestore"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip77"
)

func main() {
	ctx := context.Background()
	db := &slicestore.SliceStore{}
	db.Init()

	sk := nostr.GeneratePrivateKey()
	local := eventstore.RelayWrapper{Store: db}

	for {
		for i := 0; i < 20; i++ {
			{
				evt := nostr.Event{
					Kind:      1,
					Content:   fmt.Sprintf("same old hello %d", i),
					CreatedAt: nostr.Timestamp(i),
					Tags:      nostr.Tags{},
				}
				evt.Sign(sk)
				db.SaveEvent(ctx, &evt)
			}

			{
				evt := nostr.Event{
					Kind:      1,
					Content:   fmt.Sprintf("custom hello %d", i),
					CreatedAt: nostr.Now(),
					Tags:      nostr.Tags{},
				}
				evt.Sign(sk)
				db.SaveEvent(ctx, &evt)
			}
		}

		err := nip77.NegentropySync(ctx,
			local, "ws://localhost:7777", nostr.Filter{}, nip77.Both)
		if err != nil {
			panic(err)
		}

		data, err := local.QuerySync(ctx, nostr.Filter{})
		if err != nil {
			panic(err)
		}

		fmt.Println("total local events:", len(data))
		time.Sleep(time.Second * 10)
	}
}
