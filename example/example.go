package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func utilTrimString(s string) string {
	return strings.Trim(s, "\r\n\t ")
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	// connect to relay
	url := "wss://nostr.zebedee.cloud"
	// url := "wss://relay.nostr.net.in"
	relay, err := nostr.RelayConnect(ctx, url)
	if err != nil {
		panic(err)
	}

	reader := bufio.NewReader(os.Stdin)

	_, _ = fmt.Fprintf(os.Stderr, "using %s\n----\nexample subscription for three most recent notes mentioning user\npaste npub key: ", url)
	npub, _ := reader.ReadString('\n')
	npub = utilTrimString(npub)

	// create filters
	var filters nostr.Filters
	if _, v, err := nip19.Decode(npub); err == nil {
		t := make(map[string][]string)
		// making a "p" tag for the above public key.
		// this filters for messages tagged with the user, mainly replies.
		t["p"] = []string{v.(string)}
		filters = []nostr.Filter{{
			Kinds: []int{1},
			Tags:  t,
			// limit = 3, get the three most recent notes
			Limit: 3,
		}}
	} else {
		panic("not a valid npub!")
	}

	// create a subscription and submit to relay
	// results will be returned on the sub.Events channel
	sub := relay.Subscribe(ctx, filters)

	// we will append the returned events to this slice
	evs := make([]nostr.Event, 0)

	go func() {
		<-sub.EndOfStoredEvents
		cancel()
	}()
	for ev := range sub.Events {
		evs = append(evs, *ev)
	}

	filename := "example_output.json"
	if f, err := os.Create(filename); err == nil {
		fmt.Fprintf(os.Stderr, "returned events saved to %s\n", filename)
		// encode the returned events in a file
		enc := json.NewEncoder(f)
		enc.SetIndent("", " ")
		enc.Encode(evs)
		f.Close()
	} else {
		panic(err)
	}

	fmt.Fprintf(os.Stderr, "----\nexample publication of note.\npaste nsec key (leave empty to autogenerate): ")
	nsec, _ := reader.ReadString('\n')
	nsec = utilTrimString(nsec)

	var sk string
	ev := nostr.Event{}
	if _, s, e := nip19.Decode(nsec); e == nil {
		sk = s.(string)
	} else {
		sk = nostr.GeneratePrivateKey()
	}
	if pub, e := nostr.GetPublicKey(sk); e == nil {
		ev.PubKey = pub
		if npub, e := nip19.EncodePublicKey(pub); e == nil {
			fmt.Fprintln(os.Stderr, "using:", npub)
		}
	} else {
		panic(e)
	}

	ev.CreatedAt = time.Now()
	ev.Kind = 1
	var content string
	fmt.Fprintln(os.Stderr, "enter content of note, ending with an empty newline (ctrl+d):")
	for {
		curContent, _ := reader.ReadString('\n')
		curContent = utilTrimString(curContent)
		if curContent == "" {
			break
		}

		content += curContent
	}
	ev.Content = strings.TrimSpace(content)
	ev.Sign(sk)
	for _, url := range []string{url} {
		ctx := context.WithValue(context.Background(), "url", url)
		relay, e := nostr.RelayConnect(ctx, url)
		if e != nil {
			fmt.Println(e)
			continue
		}

		status, e := relay.Publish(ctx, ev)
		fmt.Println("posting to: ", url, status, e)
	}
}
