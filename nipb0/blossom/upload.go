package blossom

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"

	"github.com/nbd-wtf/go-nostr"
)

// UploadFile uploads a file to the media server
func (c *Client) UploadFile(ctx context.Context, filePath string) (*BlobDescriptor, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filePath, err)
	}
	defer file.Close()

	sha := sha256.New()
	size, err := io.Copy(sha, file)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
	}
	hash := sha.Sum(nil)

	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to reset file position: %w", err)
	}

	contentType := mime.TypeByExtension(filepath.Ext(filePath))

	bd := BlobDescriptor{}
	err = c.httpCall(ctx, "PUT", "upload", contentType, func() string {
		return c.authorizationHeader(ctx, func(evt *nostr.Event) {
			evt.Tags = append(evt.Tags, nostr.Tag{"t", "upload"})
			evt.Tags = append(evt.Tags, nostr.Tag{"x", hex.EncodeToString(hash[:])})
		})
	}, file, size, &bd)
	if err != nil {
		return nil, fmt.Errorf("failed to upload %s: %w", filePath, err)
	}

	return &bd, nil
}
