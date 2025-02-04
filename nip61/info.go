package nip61

import (
	"context"
	"slices"

	"github.com/elnosh/gonuts/cashu"
	"github.com/nbd-wtf/go-nostr"
)

type Info struct {
	PublicKey string
	Mints     []string
	Relays    []string
}

func (zi *Info) ToEvent(ctx context.Context, kr nostr.Keyer, evt *nostr.Event) error {
	evt.CreatedAt = nostr.Now()
	evt.Kind = 10019

	evt.Tags = make(nostr.Tags, 0, len(zi.Mints)+len(zi.Relays)+1)
	for _, mint := range zi.Mints {
		evt.Tags = append(evt.Tags, nostr.Tag{"mint", mint})
	}
	for _, url := range zi.Relays {
		evt.Tags = append(evt.Tags, nostr.Tag{"relay", url})
	}
	if zi.PublicKey != "" {
		evt.Tags = append(evt.Tags, nostr.Tag{"pubkey", zi.PublicKey})
	}

	if err := kr.SignEvent(ctx, evt); err != nil {
		return err
	}

	return nil
}

func (zi *Info) ParseEvent(evt *nostr.Event) error {
	zi.Mints = make([]string, 0)
	for _, tag := range evt.Tags {
		if len(tag) < 2 {
			continue
		}

		switch tag[0] {
		case "mint":
			if len(tag) == 2 || slices.Contains(tag[2:], cashu.Sat.String()) {
				url, _ := nostr.NormalizeHTTPURL(tag[1])
				zi.Mints = append(zi.Mints, url)
			}
		case "relay":
			zi.Relays = append(zi.Relays, tag[1])
		case "pubkey":
			if nostr.IsValidPublicKey(tag[1]) {
				zi.PublicKey = tag[1]
			}
		}
	}

	return nil
}
