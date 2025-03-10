package keyer

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip44"
	"github.com/nbd-wtf/go-nostr/nip49"
)

var _ nostr.Keyer = (*EncryptedKeySigner)(nil)

// EncryptedKeySigner is a signer that must ask the user for a password before every operation.
// It stores the private key in encrypted form (NIP-49) and uses a callback to request the password
// when needed for operations.
type EncryptedKeySigner struct {
	ncryptsec string
	pk        string
	callback  func(context.Context) string
}

// GetPublicKey returns the public key associated with this signer.
// If the public key is not cached, it will decrypt the private key using the password
// callback to derive the public key.
func (es *EncryptedKeySigner) GetPublicKey(ctx context.Context) (string, error) {
	if es.pk != "" {
		return es.pk, nil
	}
	password := es.callback(ctx)
	key, err := nip49.Decrypt(es.ncryptsec, password)
	if err != nil {
		return "", err
	}
	pk, err := nostr.GetPublicKey(key)
	if err != nil {
		return "", err
	}
	es.pk = pk
	return pk, nil
}

// SignEvent signs the provided event by first decrypting the private key
// using the password callback, then signing the event with the decrypted key.
func (es *EncryptedKeySigner) SignEvent(ctx context.Context, evt *nostr.Event) error {
	password := es.callback(ctx)
	sk, err := nip49.Decrypt(es.ncryptsec, password)
	if err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}
	es.pk = evt.PubKey
	return evt.Sign(sk)
}

// Encrypt encrypts a plaintext message for a recipient using NIP-44.
// It first decrypts the private key using the password callback.
func (es EncryptedKeySigner) Encrypt(ctx context.Context, plaintext string, recipient string) (c64 string, err error) {
	password := es.callback(ctx)
	sk, err := nip49.Decrypt(es.ncryptsec, password)
	if err != nil {
		return "", fmt.Errorf("invalid password: %w", err)
	}
	ck, err := nip44.GenerateConversationKey(recipient, sk)
	if err != nil {
		return "", err
	}
	return nip44.Encrypt(plaintext, ck)
}

// Decrypt decrypts a base64-encoded ciphertext from a sender using NIP-44.
// It first decrypts the private key using the password callback.
func (es EncryptedKeySigner) Decrypt(ctx context.Context, base64ciphertext string, sender string) (plaintext string, err error) {
	password := es.callback(ctx)
	sk, err := nip49.Decrypt(es.ncryptsec, password)
	if err != nil {
		return "", fmt.Errorf("invalid password: %w", err)
	}
	ck, err := nip44.GenerateConversationKey(sender, sk)
	if err != nil {
		return "", err
	}
	return nip44.Encrypt(plaintext, ck)
}
