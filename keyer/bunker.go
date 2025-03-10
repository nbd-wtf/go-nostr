package keyer

import (
	"context"
	"errors"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip46"
)

var _ nostr.Keyer = (*BunkerSigner)(nil)

// BunkerSigner is a signer that delegates operations to a remote bunker using NIP-46.
// It communicates with the bunker for all cryptographic operations rather than
// handling the private key locally.
type BunkerSigner struct {
	bunker *nip46.BunkerClient
}

// NewBunkerSignerFromBunkerClient creates a new BunkerSigner from an existing BunkerClient.
func NewBunkerSignerFromBunkerClient(bc *nip46.BunkerClient) BunkerSigner {
	return BunkerSigner{bc}
}

// GetPublicKey retrieves the public key from the remote bunker.
// It uses a timeout to prevent hanging indefinitely.
func (bs BunkerSigner) GetPublicKey(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeoutCause(ctx, time.Second*30, errors.New("get_public_key took too long"))
	defer cancel()
	pk, err := bs.bunker.GetPublicKey(ctx)
	if err != nil {
		return "", err
	}
	return pk, nil
}

// SignEvent sends the event to the remote bunker for signing.
// It uses a timeout to prevent hanging indefinitely.
func (bs BunkerSigner) SignEvent(ctx context.Context, evt *nostr.Event) error {
	ctx, cancel := context.WithTimeoutCause(ctx, time.Second*30, errors.New("sign_event took too long"))
	defer cancel()
	return bs.bunker.SignEvent(ctx, evt)
}

// Encrypt encrypts a plaintext message for a recipient using the remote bunker.
func (bs BunkerSigner) Encrypt(ctx context.Context, plaintext string, recipient string) (string, error) {
	return bs.bunker.NIP44Encrypt(ctx, recipient, plaintext)
}

// Decrypt decrypts a base64-encoded ciphertext from a sender using the remote bunker.
func (bs BunkerSigner) Decrypt(ctx context.Context, base64ciphertext string, sender string) (plaintext string, err error) {
	return bs.bunker.NIP44Encrypt(ctx, sender, base64ciphertext)
}
