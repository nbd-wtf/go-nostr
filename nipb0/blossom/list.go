package blossom

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

// List retrieves a list of blobs from a specific pubkey
func (c *Client) List(ctx context.Context) ([]BlobDescriptor, error) {
	pubkey, err := c.signer.GetPublicKey(ctx)
	if !nostr.IsValidPublicKey(pubkey) {
		return nil, fmt.Errorf("pubkey %s is not valid", pubkey)
	}

	bds := make([]BlobDescriptor, 0, 100)
	err = c.httpCall(ctx, "GET", "list/"+pubkey, "", func() string {
		return c.authorizationHeader(ctx, func(evt *nostr.Event) {
			evt.Tags = append(evt.Tags, nostr.Tag{"t", "list"})
		})
	}, nil, 0, &bds)
	if err != nil {
		return nil, fmt.Errorf("failed to list blobs: %w", err)
	}

	return bds, nil
}
