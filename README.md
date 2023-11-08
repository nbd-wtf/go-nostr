[![Run Tests](https://github.com/nbd-wtf/go-nostr/actions/workflows/test.yml/badge.svg)](https://github.com/nbd-wtf/go-nostr/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/nbd-wtf/go-nostr.svg)](https://pkg.go.dev/github.com/nbd-wtf/go-nostr)
[![Go Report Card](https://goreportcard.com/badge/github.com/nbd-wtf/go-nostr)](https://goreportcard.com/report/github.com/nbd-wtf/go-nostr)

<a href="https://nbd.wtf"><img align="right" height="196" src="https://user-images.githubusercontent.com/1653275/194609043-0add674b-dd40-41ed-986c-ab4a2e053092.png" /></a>

go-nostr
========

A set of useful things for [Nostr Protocol](https://github.com/nostr-protocol/nostr) implementations.

```bash
go get github.com/nbd-wtf/go-nostr
```

### Generating a key

``` go
package main

import (
    "fmt"

    "github.com/nbd-wtf/go-nostr"
    "github.com/nbd-wtf/go-nostr/nip19"
)

func main() {
    sk := nostr.GeneratePrivateKey()
    pk, _ := nostr.GetPublicKey(sk)
    nsec, _ := nip19.EncodePrivateKey(sk)
    npub, _ := nip19.EncodePublicKey(pk)

    fmt.Println("sk:", sk)
    fmt.Println("pk:", pk)
    fmt.Println(nsec)
    fmt.Println(npub)
}
```

### Subscribing to a single relay

``` go
ctx := context.Background()
relay, err := nostr.RelayConnect(ctx, "wss://nostr.zebedee.cloud")
if err != nil {
	panic(err)
}

npub := "npub1422a7ws4yul24p0pf7cacn7cghqkutdnm35z075vy68ggqpqjcyswn8ekc"

var filters nostr.Filters
if _, v, err := nip19.Decode(npub); err == nil {
	pub := v.(string)
	filters = []nostr.Filter{{
		Kinds:   []int{nostr.KindTextNote},
		Authors: []string{pub},
		Limit:   1,
	}}
} else {
	panic(err)
}

ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
defer cancel()

sub, err := relay.Subscribe(ctx, filters)
if err != nil {
	panic(err)
}

for ev := range sub.Events {
	// handle returned event.
	// channel will stay open until the ctx is cancelled (in this case, context timeout)
	fmt.Println(ev.ID)
}
```

### Publishing to two relays

``` go
sk := nostr.GeneratePrivateKey()
pub, _ := nostr.GetPublicKey(sk)

ev := nostr.Event{
	PubKey:    pub,
	CreatedAt: nostr.Now(),
	Kind:      nostr.KindTextNote,
	Tags:      nil,
	Content:   "Hello World!",
}

// calling Sign sets the event ID field and the event Sig field
ev.Sign(sk)

// publish the event to two relays
ctx := context.Background()
for _, url := range []string{"wss://nostr.zebedee.cloud", "wss://nostr-pub.wellorder.net"} {
	relay, err := nostr.RelayConnect(ctx, url)
	if err != nil {
		fmt.Println(err)
		continue
	}
	_, err = relay.Publish(ctx, ev)
	if err != nil {
		fmt.Println(err)
		continue
	}

	fmt.Printf("published to %s\n", url)
}
```

### Authenticating with NIP-42

For this section, the user needs access to a relay implementing NIP-42.
E.g., https://github.com/fiatjaf/relayer with a relay implementing the relayer.Auther interface.

``` go
func main() {
	url := "ws://localhost:7447"

	sk := nostr.GeneratePrivateKey()

	relay, err := nostr.RelayConnect(context.Background(), url,
        nostr.WithAuthHandler(func(ctx context.Context, authEvent *Event) (ok bool) {
            authEvent.Sign(sk)
        }),
    )
	if err != nil {
		panic(err)
	}
}
```

### Example script

```
go run example/example.go
```

## Warning: risk of goroutine bloat (if used incorrectly)

Remember to cancel subscriptions, either by calling `.Unsub()` on them or ensuring their `context.Context` will be canceled at some point.
If you don't do that they will keep creating a new goroutine for every new event that arrives and if you have stopped listening on the
`sub.Events` channel that will cause chaos and doom in your program.
