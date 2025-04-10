package sdk

import (
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestPrepareNoteEvent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantTags nostr.Tags
		want     string
	}{
		{
			name:     "plain text",
			content:  "hello world",
			wantTags: nostr.Tags{},
			want:     "hello world",
		},
		{
			name:    "with nostr: prefix, url and hashtag",
			content: "hello nostr:npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6 please visit https://banana.com/ and get your free #banana",
			wantTags: nostr.Tags{
				{"p", "82341f882b6eabcd2ba7f1ef90aad961cf074af15b9ef44a09f9d2a8fbfbe6a2"},
			},
			want: "hello nostr:npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6 please visit https://banana.com/ and get your free #banana",
		},
		{
			name:    "with bare npub and bare url",
			content: "hello npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6 please visit banana.com",
			wantTags: nostr.Tags{
				{"p", "82341f882b6eabcd2ba7f1ef90aad961cf074af15b9ef44a09f9d2a8fbfbe6a2"},
			},
			want: "hello nostr:npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6 please visit banana.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := &nostr.Event{
				Content: tt.content,
				Tags:    nostr.Tags{},
			}

			PrepareNoteEvent(evt)
			require.Equal(t, tt.want, evt.Content)
			require.Equal(t, tt.wantTags, evt.Tags)
		})
	}
}
