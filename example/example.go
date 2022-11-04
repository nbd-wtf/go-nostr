package main

import (
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

// some nostr relay in the wild
var relayURL = "wss://nostr-relay.wlvs.space"

func main() {
	// create key pair
	secretKey := nostr.GeneratePrivateKey()
	publicKey, err := nostr.GetPublicKey(secretKey)
	if err != nil {
		fmt.Printf("error with GetPublicKey(): %s\n", err)
		return
	}
	fmt.Printf("Our Pubkey: %s\n\n", publicKey)
	fmt.Printf("Lets send and receive 4 events...\n\n")

	// subscribe to relay
	pool := nostr.NewRelayPool()
	pool.SecretKey = &secretKey

	// add a nostr relay to our pool
	err = pool.Add(relayURL, nostr.SimplePolicy{Read: true, Write: true})
	if err != nil {
		fmt.Printf("error calling Add(): %s\n", err.Error())
	}

	// subscribe to relays in our pool, with filtering
	sub := pool.Sub(nostr.Filters{
		{
			Authors: []string{publicKey},
			Kinds:   []int{nostr.KindTextNote},
		},
	})

	// listen for events from our subscriptions
	go func() {
		for event := range sub.UniqueEvents {
			fmt.Printf("Received Event: %+v\n\n", event)
		}
	}()

	// create and publish events
	go func() {
		for {
			content := fmt.Sprintf("henlo world at time: %s", time.Now().String())
			event, statuses, err := pool.PublishEvent(&nostr.Event{
				CreatedAt: time.Now(),
				Kind:      nostr.KindTextNote,
				Tags:      make(nostr.Tags, 0),
				Content:   content,
			})
			if err != nil {
				fmt.Printf("error calling PublishEvent(): %s\n", err.Error())
			}

			StatusProcess(event, statuses)
			// sleep between publishing events
			time.Sleep(time.Second * 5)
		}
	}()

	// after 20 seconds, unsubscribe from our pool and terminate program
	time.Sleep(20 * time.Second)
	fmt.Println("unsubscribing from nostr subscription")
	sub.Unsub()
}

// handle events from out publish events
func StatusProcess(event *nostr.Event, statuses chan nostr.PublishStatus) {
	for status := range statuses {
		switch status.Status {
		case nostr.PublishStatusSent:
			fmt.Printf("Sent event with id %s to '%s'.\n", event.ID, status.Relay)
			return
		case nostr.PublishStatusFailed:
			fmt.Printf("Failed to send event with id %s to '%s'.\n", event.ID, status.Relay)
			return
		case nostr.PublishStatusSucceeded:
			fmt.Printf("Event with id %s seen on '%s'.\n", event.ID, status.Relay)
			return
		}
	}
}
