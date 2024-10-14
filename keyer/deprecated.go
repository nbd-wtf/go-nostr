package keyer

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
)

// Deprecated: use nostr.Keyer instead
type Keyer interface {
	Signer
	Cipher
}

// Deprecated: use nostr.Signer instead
type Signer interface {
	GetPublicKey(context.Context) (string, error)
	SignEvent(context.Context, *nostr.Event) error
}

// Deprecated: use nostr.Cipher instead
type Cipher interface {
	Encrypt(ctx context.Context, plaintext string, recipientPublicKey string) (base64ciphertext string, err error)
	Decrypt(ctx context.Context, base64ciphertext string, senderPublicKey string) (plaintext string, err error)
}
