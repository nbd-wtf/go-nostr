<a href="https://nbd.wtf"><img align="right" height="196" src="https://user-images.githubusercontent.com/1653275/194609043-0add674b-dd40-41ed-986c-ab4a2e053092.png" /></a>

go-nostr
========

A set of useful things for [Nostr Protocol](https://github.com/nostr-protocol/nostr) implementations.

<a href="https://godoc.org/github.com/nbd-wtf/go-nostr"><img src="https://img.shields.io/badge/api-reference-blue.svg?style=flat-square" alt="GoDoc"></a>

### Generating a key

``` go
sk, _ := nostr.GenerateKey()
pk, _ := nostr.GetPublicKey(sk)
nsec, _ := nip19.EncodePrivateKey(sk)
npub, _ := nip19.EncodePublicKey(pk)

fmt.Println("sk:", sk)
fmt.Println("pk:", nostr.GetPublicKey(sk))
fmt.Println(nsec)
fmt.Println(npub)
```

### Subscribing to a single relay

``` go
relay, err := nostr.RelayConnect(context.Background(), "wss://nostr.zebedee.cloud")
if err != nil {
	panic(err)
}

npub := "npub1422a7ws4yul24p0pf7cacn7cghqkutdnm35z075vy68ggqpqjcyswn8ekc"

var filters nostr.Filters
if _, v, err := nip19.Decode(npub); err == nil {
	pub := v.(string)
	filters = []nostr.Filter{{
		Kinds:   []int{1},
		Authors: []string{pub},
		Limit:   1,
	}}
} else {
	panic(err)
}

sub := relay.Subscribe(context.Background(), filters)

go func() {
	<-sub.EndOfStoredEvents
	// handle end of stored events (EOSE, see NIP-15)
}()

for ev := range sub.Events {
	// handle returned event.
	// channel will stay open until sub.Unsub() is called
}
```

### Publishing to two relays

``` go
sk := nostr.GeneratePrivateKey()
pub, _ := nostr.GetPublicKey(sk)

ev := nostr.Event{
	PubKey:    pub,
	CreatedAt: time.Now(),
	Kind:      1,
	Tags:      nil,
	Content:   "Hello World!",
}

// calling Sign sets the event ID field and the event Sig field
ev.Sign(sk)

// publish the event to two relays
for _, url := range []string{"wss://nostr.zebedee.cloud", "wss://nostr-pub.wellorder.net"} {
	relay, e := nostr.RelayConnect(context.Background(), url)
	if e != nil {
		fmt.Println(e)
		continue
	}
	fmt.Println("published to ", url, relay.Publish(context.Background(), ev))
}
```

### Example script

```
go run example/example.go
```
