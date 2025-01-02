package sdk

import (
	"fmt"
	"slices"
	"testing"

	"github.com/nbd-wtf/go-nostr"
)

func TestParseReferences(t *testing.T) {
	evt := nostr.Event{
		Tags: nostr.Tags{
			{"p", "c9d556c6d3978d112d30616d0d20aaa81410e3653911dd67787b5aaf9b36ade8", "wss://nostr.com"},
			{"e", "a84c5de86efc2ec2cff7bad077c4171e09146b633b7ad117fffe088d9579ac33", "wss://other.com", "reply"},
			{"e", "31d7c2875b5fc8e6f9c8f9dc1f84de1b6b91d1947ea4c59225e55c325d330fa8", ""},
		},
		Content: "hello #[0], have you seen #[2]? it was made by nostr:nprofile1qqsvc6ulagpn7kwrcwdqgp797xl7usumqa6s3kgcelwq6m75x8fe8yc5usxdg on nostr:nevent1qqsvc6ulagpn7kwrcwdqgp797xl7usumqa6s3kgcelwq6m75x8fe8ychxp5v4! broken #[3]",
	}

	expected := []Reference{
		{
			Text:  "#[0]",
			Start: 6,
			End:   10,
			Profile: &nostr.ProfilePointer{
				PublicKey: "c9d556c6d3978d112d30616d0d20aaa81410e3653911dd67787b5aaf9b36ade8",
				Relays:    []string{"wss://nostr.com"},
			},
		},
		{
			Text:  "#[2]",
			Start: 26,
			End:   30,
			Event: &nostr.EventPointer{
				ID:     "31d7c2875b5fc8e6f9c8f9dc1f84de1b6b91d1947ea4c59225e55c325d330fa8",
				Relays: []string{},
			},
		},
		{
			Text:  "nostr:nprofile1qqsvc6ulagpn7kwrcwdqgp797xl7usumqa6s3kgcelwq6m75x8fe8yc5usxdg",
			Start: 47,
			End:   123,
			Profile: &nostr.ProfilePointer{
				PublicKey: "cc6b9fea033f59c3c39a0407c5f1bfee439b077508d918cfdc0d6fd431d39393",
				Relays:    []string{},
			},
		},
		{
			Text:  "nostr:nevent1qqsvc6ulagpn7kwrcwdqgp797xl7usumqa6s3kgcelwq6m75x8fe8ychxp5v4",
			Start: 127,
			End:   201,
			Event: &nostr.EventPointer{
				ID:     "cc6b9fea033f59c3c39a0407c5f1bfee439b077508d918cfdc0d6fd431d39393",
				Relays: []string{},
				Author: "",
			},
		},
	}

	got := slices.Collect(ParseReferences(evt))

	if len(got) != len(expected) {
		t.Errorf("got %d references, expected %d", len(got), len(expected))
	}

	for i, g := range got {
		e := expected[i]
		if g.Text != e.Text {
			t.Errorf("%d: got text %s, expected %s", i, g.Text, e.Text)
		}

		if g.Start != e.Start {
			t.Errorf("%d: got start %d, expected %d", i, g.Start, e.Start)
		}

		if g.End != e.End {
			t.Errorf("%d: got end %d, expected %d", i, g.End, e.End)
		}

		if (g.Entity == nil && e.Entity != nil) ||
			(g.Event == nil && e.Event != nil) ||
			(g.Profile == nil && e.Profile != nil) {
			t.Errorf("%d: got some unexpected nil", i)
		}

		if g.Profile != nil && (g.Profile.PublicKey != e.Profile.PublicKey ||
			len(g.Profile.Relays) != len(e.Profile.Relays) ||
			(len(g.Profile.Relays) > 0 && g.Profile.Relays[0] != e.Profile.Relays[0])) {
			t.Errorf("%d: profile value is wrong", i)
		}

		if g.Event != nil && (g.Event.ID != e.Event.ID ||
			g.Event.Author != e.Event.Author ||
			len(g.Event.Relays) != len(e.Event.Relays) ||
			(len(g.Event.Relays) > 0 && g.Event.Relays[0] != e.Event.Relays[0])) {
			fmt.Println(g.Event.ID, g.Event.Relays, len(g.Event.Relays), g.Event.Relays[0] == "")
			fmt.Println(e.Event.Relays, len(e.Event.Relays))
			t.Errorf("%d: event value is wrong", i)
		}

		if g.Entity != nil && (g.Entity.PublicKey != e.Entity.PublicKey ||
			g.Entity.Identifier != e.Entity.Identifier ||
			g.Entity.Kind != e.Entity.Kind ||
			len(g.Entity.Relays) != len(g.Entity.Relays)) {
			t.Errorf("%d: entity value is wrong", i)
		}
	}
}
