package nip27

import (
	"slices"
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestParseReferences(t *testing.T) {
	evt := nostr.Event{
		Tags: nostr.Tags{
			{"p", "cc6b9fea033f59c3c39a0407c5f1bfee439b077508d918cfdc0d6fd431d39393", "wss://xawr.com"},
			{"e", "a84c5de86efc2ec2cff7bad077c4171e09146b633b7ad117fffe088d9579ac33", "wss://other.com", "reply"},
			{"e", "cc6b9fea033f59c3c39a0407c5f1bfee439b077508d918cfdc0d6fd431d39393", "wss://nasdj.com"},
		},
		Content: "hello, nostr:nprofile1qqsvc6ulagpn7kwrcwdqgp797xl7usumqa6s3kgcelwq6m75x8fe8yc5usxdg wrote nostr:nevent1qqsvc6ulagpn7kwrcwdqgp797xl7usumqa6s3kgcelwq6m75x8fe8ychxp5v4!",
	}

	expected := []Reference{
		{
			Text:  "nostr:nprofile1qqsvc6ulagpn7kwrcwdqgp797xl7usumqa6s3kgcelwq6m75x8fe8yc5usxdg",
			Start: 7,
			End:   83,
			Pointer: nostr.ProfilePointer{
				PublicKey: "cc6b9fea033f59c3c39a0407c5f1bfee439b077508d918cfdc0d6fd431d39393",
				Relays:    []string{"wss://xawr.com"},
			},
		},
		{
			Text:  "nostr:nevent1qqsvc6ulagpn7kwrcwdqgp797xl7usumqa6s3kgcelwq6m75x8fe8ychxp5v4",
			Start: 90,
			End:   164,
			Pointer: nostr.EventPointer{
				ID:     "cc6b9fea033f59c3c39a0407c5f1bfee439b077508d918cfdc0d6fd431d39393",
				Relays: []string{"wss://nasdj.com"},
				Author: "",
			},
		},
	}

	got := slices.Collect(ParseReferences(evt))

	require.EqualValues(t, expected, got)
}
