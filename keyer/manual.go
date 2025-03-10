package keyer

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
)

var _ nostr.Keyer = (*ManualSigner)(nil)

// ManualSigner is a signer that delegates all operations to user-provided functions.
// It can be used when an app wants to ask the user or some custom server to manually provide a
// signed event or an encrypted or decrypted payload by copy-and-paste, for example, or when the
// app wants to implement custom signing logic.
type ManualSigner struct {
	// ManualGetPublicKey is called when the public key is needed
	ManualGetPublicKey func(context.Context) (string, error)

	// ManualSignEvent is called when an event needs to be signed
	ManualSignEvent func(context.Context, *nostr.Event) error

	// ManualEncrypt is called when a message needs to be encrypted
	ManualEncrypt func(ctx context.Context, plaintext string, recipientPublicKey string) (base64ciphertext string, err error)

	// ManualDecrypt is called when a message needs to be decrypted
	ManualDecrypt func(ctx context.Context, base64ciphertext string, senderPublicKey string) (plaintext string, err error)
}

// SignEvent delegates event signing to the ManualSignEvent function.
func (ms ManualSigner) SignEvent(ctx context.Context, evt *nostr.Event) error {
	return ms.ManualSignEvent(ctx, evt)
}

// GetPublicKey delegates public key retrieval to the ManualGetPublicKey function.
func (ms ManualSigner) GetPublicKey(ctx context.Context) (string, error) {
	return ms.ManualGetPublicKey(ctx)
}

// Encrypt delegates encryption to the ManualEncrypt function.
func (ms ManualSigner) Encrypt(ctx context.Context, plaintext string, recipient string) (c64 string, err error) {
	return ms.ManualEncrypt(ctx, plaintext, recipient)
}

// Decrypt delegates decryption to the ManualDecrypt function.
func (ms ManualSigner) Decrypt(ctx context.Context, base64ciphertext string, sender string) (plaintext string, err error) {
	return ms.ManualDecrypt(ctx, base64ciphertext, sender)
}
