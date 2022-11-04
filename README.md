<a href="https://nbd.wtf"><img align="right" height="196" src="https://user-images.githubusercontent.com/1653275/194609043-0add674b-dd40-41ed-986c-ab4a2e053092.png" /></a>

go-nostr
========

A set of useful things for [Nostr Protocol](https://github.com/nostr-protocol/nostr) implementations.

<a href="https://godoc.org/github.com/nbd-wtf/go-nostr"><img src="https://img.shields.io/badge/api-reference-blue.svg?style=flat-square" alt="GoDoc"></a>


### Subscribing to a set of relays

```go
pool := nostr.NewRelayPool()

pool.Add("wss://relay.nostr.com/",  nostr.SimplePolicy{Read: true, Write: true})
pool.Add("wss://nostrrelay.example.com/",  nostr.SimplePolicy{Read: true, Write: true})

for notice := range pool.Notices {
	log.Printf("%s has sent a notice: '%s'\n", notice.Relay, notice.Message)
}
```

### Listening for events

```go
sub := pool.Sub(nostr.Filters{
	{
		Authors: []string{"0ded86bf80c76847320b16f22b7451c08169434837a51ad5fe3b178af6c35f5d"},
		Kinds:   []int{nostr.KindTextNote}, // or {1}
	},
})

go func() {
	for event := range sub.UniqueEvents {
		log.Print(event)
	}
}()

time.Sleep(5 * time.Second)
sub.Unsub()
```

### Publishing an event

```go
secretKey := "3f06a81e0a0c2ad34ee9df2a30d87a810da9e3c3881f780755ace5e5e64d30a7"

pool.SecretKey = &secretKey

event, statuses, _ := pool.PublishEvent(&nostr.Event{
	CreatedAt: time.Now(),
	Kind:      nostr.KindTextNote,
	Tags:      make(nostr.Tags, 0),
	Content:   "hello",
})

log.Print(event.PubKey)
log.Print(event.ID)
log.Print(event.Sig)

for status := range statuses {
	switch status.Status {
	case nostr.PublishStatusSent:
		fmt.Printf("Sent event %s to '%s'.\n", event.ID, status.Relay)
	case nostr.PublishStatusFailed:
		fmt.Printf("Failed to send event %s to '%s'.\n", event.ID, status.Relay)
	case nostr.PublishStatusSucceeded:
		fmt.Printf("Event seen %s on '%s'.\n", event.ID, status.Relay)
	}
}
```

### Generating a key

``` go
sk, _ := nostr.GenerateKey()

fmt.Println("sk:", sk)
fmt.Println("pk:", nostr.GetPublicKey(sk))
```

### Example Program

```
go run example/example.go
```
