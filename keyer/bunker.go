package keyer

import (
	"context"
	"errors"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip46"
)

// BunkerSigner is a signer that asks a bunker using NIP-46 every time it needs to do an operation.
type BunkerSigner struct {
	bunker *nip46.BunkerClient
}

func NewBunkerSignerFromBunkerClient(bc *nip46.BunkerClient) BunkerSigner {
	return BunkerSigner{bc}
}

func (bs BunkerSigner) GetPublicKey(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeoutCause(ctx, time.Second*30, errors.New("get_public_key took too long"))
	defer cancel()
	pk, err := bs.bunker.GetPublicKey(ctx)
	if err != nil {
		return "", err
	}
	return pk, nil
}

func (bs BunkerSigner) SignEvent(ctx context.Context, evt *nostr.Event) error {
	ctx, cancel := context.WithTimeoutCause(ctx, time.Second*30, errors.New("sign_event took too long"))
	defer cancel()
	return bs.bunker.SignEvent(ctx, evt)
}

func (bs BunkerSigner) Encrypt(ctx context.Context, plaintext string, recipient string) (string, error) {
	return bs.bunker.NIP44Encrypt(ctx, recipient, plaintext)
}

func (bs BunkerSigner) Decrypt(ctx context.Context, base64ciphertext string, sender string) (plaintext string, err error) {
	return bs.bunker.NIP44Encrypt(ctx, sender, base64ciphertext)
}
