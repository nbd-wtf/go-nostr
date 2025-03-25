//go:build !js

package nip96

import (
	"context"
	"os"
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpload(t *testing.T) {
	img, err := os.Open("./testdata/image.png")
	require.NoError(t, err)

	defer img.Close()

	ctx := context.Background()
	resp, err := Upload(ctx, UploadRequest{
		Host: "https://nostr.build/api/v2/nip96/upload",
		// Host:        "https://nostrcheck.me/api/v2/media",
		// Host:        "https://nostrage.com/api/v2/media",
		SK:          nostr.GeneratePrivateKey(),
		SignPayload: true,
		File:        img,
		Filename:    "ostrich.png",
		Caption:     "nostr ostrich",
		ContentType: "image/png",
		NoTransform: true,
	})
	assert.NoError(t, err)

	t.Logf("resp: %#v\n", *resp)
	// nip96_test.go:28: resp: nip96.UploadResponse{Status:"success", Message:"Upload successful.", ProcessingURL:"", Nip94Event:struct { Tags nostr.Tags "json:\"tags\"" }{Tags:nostr.Tags{nostr.Tag{"url", "https://image.nostr.build/4ece05f1d77c9cb97d334ba9c0301b2960640df89bf5d75d6bffadefc4355673.jpg"}, nostr.Tag{"ox", "4ece05f1d77c9cb97d334ba9c0301b2960640df89bf5d75d6bffadefc4355673"}, nostr.Tag{"x", ""}, nostr.Tag{"m", "image/jpeg"}, nostr.Tag{"dim", "1125x750"}, nostr.Tag{"bh", "LLF=kB-;yH-;-;R#t7xKEZWA#_oM"}, nostr.Tag{"blurhash", "LLF=kB-;yH-;-;R#t7xKEZWA#_oM"}, nostr.Tag{"thumb", "https://image.nostr.build/thumb/4ece05f1d77c9cb97d334ba9c0301b2960640df89bf5d75d6bffadefc4355673.jpg"}}}}
}
