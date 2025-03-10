package keyer

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip44"
	"github.com/puzpuzpuz/xsync/v3"
)

var _ nostr.Keyer = (*KeySigner)(nil)

// KeySigner is a signer that holds the private key in memory
type KeySigner struct {
	sk string
	pk string

	conversationKeys *xsync.MapOf[string, [32]byte]
}

// NewPlainKeySigner creates a new KeySigner from a private key.
// Returns an error if the private key is invalid.
func NewPlainKeySigner(sec string) (KeySigner, error) {
	pk, err := nostr.GetPublicKey(sec)
	if err != nil {
		return KeySigner{}, err
	}
	return KeySigner{sec, pk, xsync.NewMapOf[string, [32]byte]()}, nil
}

// SignEvent signs the provided event with the signer's private key.
// It sets the event's ID, PubKey, and Sig fields.
func (ks KeySigner) SignEvent(ctx context.Context, evt *nostr.Event) error { return evt.Sign(ks.sk) }

// GetPublicKey returns the public key associated with this signer.
func (ks KeySigner) GetPublicKey(ctx context.Context) (string, error) { return ks.pk, nil }

// Encrypt encrypts a plaintext message for a recipient using NIP-44.
// It caches conversation keys for efficiency in repeated operations.
func (ks KeySigner) Encrypt(ctx context.Context, plaintext string, recipient string) (string, error) {
	ck, ok := ks.conversationKeys.Load(recipient)
	if !ok {
		var err error
		ck, err = nip44.GenerateConversationKey(recipient, ks.sk)
		if err != nil {
			return "", err
		}
		ks.conversationKeys.Store(recipient, ck)
	}
	return nip44.Encrypt(plaintext, ck)
}

// Decrypt decrypts a base64-encoded ciphertext from a sender using NIP-44.
// It caches conversation keys for efficiency in repeated operations.
func (ks KeySigner) Decrypt(ctx context.Context, base64ciphertext string, sender string) (string, error) {
	ck, ok := ks.conversationKeys.Load(sender)
	if !ok {
		var err error
		ck, err = nip44.GenerateConversationKey(sender, ks.sk)
		if err != nil {
			return "", err
		}
		ks.conversationKeys.Store(sender, ck)
	}
	return nip44.Decrypt(base64ciphertext, ck)
}
