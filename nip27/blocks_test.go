package nip27

import (
	"fmt"
	"slices"
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip73"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	for i, tc := range []struct {
		content  string
		expected []Block
	}{
		{
			"hello, nostr:nprofile1qqsvc6ulagpn7kwrcwdqgp797xl7usumqa6s3kgcelwq6m75x8fe8yc5usxdg wrote nostr:nevent1qqsvc6ulagpn7kwrcwdqgp797xl7usumqa6s3kgcelwq6m75x8fe8ychxp5v4!",
			[]Block{
				{Text: "hello, ", Start: 0},
				{Text: "nostr:nprofile1qqsvc6ulagpn7kwrcwdqgp797xl7usumqa6s3kgcelwq6m75x8fe8yc5usxdg", Start: 7, Pointer: nostr.ProfilePointer{PublicKey: "cc6b9fea033f59c3c39a0407c5f1bfee439b077508d918cfdc0d6fd431d39393"}},
				{Text: " wrote ", Start: 83},
				{Text: "nostr:nevent1qqsvc6ulagpn7kwrcwdqgp797xl7usumqa6s3kgcelwq6m75x8fe8ychxp5v4", Start: 90, Pointer: nostr.EventPointer{ID: "cc6b9fea033f59c3c39a0407c5f1bfee439b077508d918cfdc0d6fd431d39393"}},
				{Text: "!", Start: 164},
			},
		},
		{
			`:wss://oa.ao; this was a relay and now here's a video -> https://videos.com/video.mp4! and some music: http://music.com/song.mp3
and a regular link: https://regular.com/page?ok=true. and now a broken link: https://kjxkxk and a broken nostr ref: nostr:nevent1qqsr0f9w78uyy09qwmjt0kv63j4l7sxahq33725lqyyp79whlfjurwspz4mhxue69uhh56nzv34hxcfwv9ehw6nyddhq0ag9xg and a fake nostr ref: nostr:llll ok but finally https://ok.com!`,
			[]Block{
				{Text: ":", Start: 0},
				{Text: "wss://oa.ao", Start: 1, Pointer: nip73.ExternalPointer{Thing: "wss://oa.ao"}},
				{Text: "; this was a relay and now here's a video -> ", Start: 12},
				{Text: "https://videos.com/video.mp4", Start: 57, Pointer: nip73.ExternalPointer{Thing: "https://videos.com/video.mp4"}},
				{Text: "! and some music: ", Start: 85},
				{Text: "http://music.com/song.mp3", Start: 103, Pointer: nip73.ExternalPointer{Thing: "http://music.com/song.mp3"}},
				{Text: "\nand a regular link: ", Start: 128},
				{Text: "https://regular.com/page?ok=true", Start: 149, Pointer: nip73.ExternalPointer{Thing: "https://regular.com/page?ok=true"}},
				{Text: ". and now a broken link: https://kjxkxk and a broken nostr ref: nostr:nevent1qqsr0f9w78uyy09qwmjt0kv63j4l7sxahq33725lqyyp79whlfjurwspz4mhxue69uhh56nzv34hxcfwv9ehw6nyddhq0ag9xg and a fake nostr ref: nostr:llll ok but finally ", Start: 181},
				{Text: "https://ok.com", Start: 405, Pointer: nip73.ExternalPointer{Thing: "https://ok.com"}},
				{Text: "!", Start: 419},
			},
		},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			require.Equal(t, tc.expected, slices.Collect(Parse(tc.content)))
		})
	}
}
