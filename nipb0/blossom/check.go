package blossom

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

// Check checks if a file exists on the media server by its hash
func (c *Client) Check(ctx context.Context, hash string) error {
	if !nostr.IsValid32ByteHex(hash) {
		return fmt.Errorf("%s is not a valid 32-byte hex string", hash)
	}

	err := c.httpCall(ctx, "HEAD", hash, "", nil, nil, 0, nil)
	if err != nil {
		return fmt.Errorf("failed to check for %s: %w", hash, err)
	}

	return nil
}
