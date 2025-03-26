package blossom

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/nbd-wtf/go-nostr"
)

// Download downloads a file from the media server by its hash
func (c *Client) Download(ctx context.Context, hash string) ([]byte, error) {
	if !nostr.IsValid32ByteHex(hash) {
		return nil, fmt.Errorf("%s is not a valid 32-byte hex string", hash)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.mediaserver+"/"+hash, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	authHeader := c.authorizationHeader(ctx, func(evt *nostr.Event) {
		evt.Tags = append(evt.Tags, nostr.Tag{"t", "get"})
		evt.Tags = append(evt.Tags, nostr.Tag{"x", hash})
	})
	req.Header.Add("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call %s for %s: %w", c.mediaserver, hash, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s is not present in %s: %d", hash, c.mediaserver, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// DownloadToFile downloads a file from the media server and saves it to the specified path
func (c *Client) DownloadToFile(ctx context.Context, hash string, filePath string) error {
	if !nostr.IsValid32ByteHex(hash) {
		return fmt.Errorf("%s is not a valid 32-byte hex string", hash)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.mediaserver+"/"+hash, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	authHeader := c.authorizationHeader(ctx, func(evt *nostr.Event) {
		evt.Tags = append(evt.Tags, nostr.Tag{"t", "get"})
		evt.Tags = append(evt.Tags, nostr.Tag{"x", hash})
	})
	req.Header.Add("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call %s for %s: %w", c.mediaserver, hash, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s is not present in %s: %d", hash, c.mediaserver, resp.StatusCode)
	}

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create file %s for %s: %w", filePath, hash, err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write to file %s for %s: %w", filePath, hash, err)
	}

	return nil
}
