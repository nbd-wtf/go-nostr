package blossom

import (
	"encoding/json"

	"github.com/nbd-wtf/go-nostr"
)

// BlobDescriptor represents metadata about a blob stored on a media server
type BlobDescriptor struct {
	URL      string          `json:"url"`
	SHA256   string          `json:"sha256"`
	Size     int             `json:"size"`
	Type     string          `json:"type"`
	Uploaded nostr.Timestamp `json:"uploaded"`
}

// String returns a JSON string representation of the BlobDescriptor
func (bd BlobDescriptor) String() string {
	j, _ := json.Marshal(bd)
	return string(j)
}
