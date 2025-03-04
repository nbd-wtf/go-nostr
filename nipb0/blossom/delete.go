package blossom

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

// Delete deletes a file from the media server by its hash
func (c *Client) Delete(ctx context.Context, hash string) error {
	err := c.httpCall(ctx, "DELETE", hash, "", func() string {
		return c.authorizationHeader(ctx, func(evt *nostr.Event) {
			evt.Tags = append(evt.Tags, nostr.Tag{"t", "delete"})
			evt.Tags = append(evt.Tags, nostr.Tag{"x", hash})
		})
	}, nil, 0, nil)
	if err != nil {
		return fmt.Errorf("failed to delete %s: %w", hash, err)
	}

	return nil
}
