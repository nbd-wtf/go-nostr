package nostr

import (
	"context"
)

// Keyer is an interface for signing events and performing cryptographic operations.
// It abstracts away the details of key management, allowing for different implementations
// such as in-memory keys, hardware wallets, or remote signing services (bunker).
type Keyer interface {
	// Signer provides event signing capabilities
	Signer

	// Cipher provides encryption and decryption capabilities (NIP-44)
	Cipher
}

// User is an entity that has a public key (although they can't sign anything).
type User interface {
	// GetPublicKey returns the public key associated with this user.
	GetPublicKey(ctx context.Context) (string, error)
}

// Signer is a User that can also sign events.
type Signer interface {
	User

	// SignEvent signs the provided event, setting its ID, PubKey, and Sig fields.
	// The context can be used for operations that may require user interaction or
	// network access, such as with remote signers.
	SignEvent(ctx context.Context, evt *Event) error
}

// Cipher is an interface for encrypting and decrypting messages with NIP-44
type Cipher interface {
	// Encrypt encrypts a plaintext message for a recipient.
	// Returns the encrypted message as a base64-encoded string.
	Encrypt(ctx context.Context, plaintext string, recipientPublicKey string) (base64ciphertext string, err error)

	// Decrypt decrypts a base64-encoded ciphertext from a sender.
	// Returns the decrypted plaintext.
	Decrypt(ctx context.Context, base64ciphertext string, senderPublicKey string) (plaintext string, err error)
}
