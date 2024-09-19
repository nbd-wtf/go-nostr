package nip96

import (
	"fmt"
	"io"
	"net/http"

	"github.com/nbd-wtf/go-nostr"
)

// UploadRequest is a NIP96 upload request.
type UploadRequest struct {
	// Host is the NIP96 server to upload to.
	Host string

	// SK is a private key used to sign the NIP-98 Auth header. If not set
	// the auth header will not be included in the upload.
	SK string
	// Optional signing of payload (file) as described in NIP-98, if enabled
	// includes `payload` tag with file's sha256 in signed event / auth header.
	SignPayload bool

	// File is the file to upload.
	File io.Reader

	// Filename is the name of the file, e.g. image.png
	Filename string

	// Caption is a loose description of the file.
	Caption string

	// Alt is a strict description text for visibility-impaired users.
	Alt string

	// MediaType is "avatar" or "banner". Informs the server if the file will be
	// used as an avatar or banner. If absent, the server will interpret it as a
	// normal upload, without special treatment.
	MediaType string

	// ContentType is the mime type such as "image/jpeg". This is just a value the
	// server can use to reject early if the mime type isn't supported.
	ContentType string

	// NoTransform set to "true" asks server not to transform the file and serve
	// the uploaded file as is, may be rejected.
	NoTransform bool

	// Expiration is a UNIX timestamp in seconds. Empty if file should be stored
	// forever. The server isn't required to honor this.
	Expiration nostr.Timestamp

	// HTTPClient is an option to provide your own HTTP Client.
	HTTPClient *http.Client
}

func (r *UploadRequest) Validate() error {
	if r.Host == "" {
		return fmt.Errorf("Host must be set")
	}

	return nil
}

// UploadResponse is a NIP96 upload response.
type UploadResponse struct {
	Status        string `json:"status"`
	Message       string `json:"message"`
	ProcessingURL string `json:"processing_url"`
	Nip94Event    struct {
		Tags    nostr.Tags `json:"tags"`
		Content string     `json:"content"`
	} `json:"nip94_event"`
}
