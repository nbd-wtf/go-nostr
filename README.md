go-nostr
========

A set of useful things for [Nostr Protocol](https://github.com/fiatjaf/nostr) implementations.

<a href="https://godoc.org/github.com/fiatjaf/go-nostr"><img src="https://img.shields.io/badge/api-reference-blue.svg?style=flat-square" alt="GoDoc"></a>


### Subscribing to a set of relays

```go
pool := relaypool.New()

pool.Add("wss://relay.nostr.com/", &relaypool.Policy{
	SimplePolicy: relaypool.SimplePolicy{Read: true, Write: true},
})
pool.Add("wss://nostrrelay.example.com/", &relaypool.Policy{
	SimplePolicy: relaypool.SimplePolicy{Read: true, Write: true},
})

for notice := range pool.Notices {
	log.Printf("%s has sent a notice: '%s'\n", notice.Relay, notice.Message)
}
```

### Listening for events

```go
kind1 := event.KindTextNote
sub := pool.Sub(filter.EventFilters{
	{
		Authors: []string{"0ded86bf80c76847320b16f22b7451c08169434837a51ad5fe3b178af6c35f5d"},
		Kind:    &kind1, // or 1
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

event, statuses, _ := pool.PublishEvent(&event.Event{
	CreatedAt: uint32(time.Now().Unix()),
	Kind:      1, // or event.KindTextNote
	Tags:      make(event.Tags, 0),
	Content:   "hello",
})

log.Print(event.PubKey)
log.Print(event.ID)
log.Print(event.Sig)

for status := range statuses {
	switch status.Status {
	case relaypool.PublishStatusSent:
		fmt.Printf("Sent event %s to '%s'.\n", event.ID, status.Relay)
	case relaypool.PublishStatusFailed:
		fmt.Printf("Failed to send event %s to '%s'.\n", event.ID, status.Relay)
	case relaypool.PublishStatusSucceeded:
		fmt.Printf("Event seen %s on '%s'.\n", event.ID, status.Relay)
	}
}
```
