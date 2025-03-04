package keyer

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip05"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/go-nostr/nip46"
	"github.com/nbd-wtf/go-nostr/nip49"
	"github.com/puzpuzpuz/xsync/v3"
)

var (
	_ nostr.Keyer = (*BunkerSigner)(nil)
	_ nostr.Keyer = (*EncryptedKeySigner)(nil)
	_ nostr.Keyer = (*KeySigner)(nil)
	_ nostr.Keyer = (*ManualSigner)(nil)
)

// SignerOptions contains configuration options for creating a new signer.
type SignerOptions struct {
	// BunkerClientSecretKey is the secret key used for the bunker client
	BunkerClientSecretKey string

	// BunkerSignTimeout is the timeout duration for bunker signing operations
	BunkerSignTimeout time.Duration

	// BunkerAuthHandler is called when authentication is needed for bunker operations
	BunkerAuthHandler func(string)

	// PasswordHandler is called when an operation needs access to the encrypted key.
	// If provided, the key will be stored encrypted and this function will be called
	// every time an operation needs access to the key so the user can be prompted.
	PasswordHandler func(context.Context) string

	// Password is used along with ncryptsec to decrypt the key.
	// If provided, the key will be decrypted and stored in plaintext.
	Password string
}

// New creates a new Keyer implementation based on the input string format.
// It supports various input formats:
// - ncryptsec: Creates an EncryptedKeySigner or KeySigner depending on options
// - NIP-46 bunker URL or NIP-05 identifier: Creates a BunkerSigner
// - nsec: Creates a KeySigner
// - hex private key: Creates a KeySigner
//
// The context is used for operations that may require network access.
// The pool is used for relay connections when needed.
// Options are used for additional pieces required for EncryptedKeySigner and BunkerSigner.
func New(ctx context.Context, pool *nostr.SimplePool, input string, opts *SignerOptions) (nostr.Keyer, error) {
	if opts == nil {
		opts = &SignerOptions{}
	}

	if strings.HasPrefix(input, "ncryptsec") {
		if opts.PasswordHandler != nil {
			return &EncryptedKeySigner{input, "", opts.PasswordHandler}, nil
		}
		sec, err := nip49.Decrypt(input, opts.Password)
		if err != nil {
			if opts.Password == "" {
				return nil, fmt.Errorf("failed to decrypt with blank password: %w", err)
			}
			return nil, fmt.Errorf("failed to decrypt with given password: %w", err)
		}
		pk, _ := nostr.GetPublicKey(sec)
		return KeySigner{sec, pk, xsync.NewMapOf[string, [32]byte]()}, nil
	} else if nip46.IsValidBunkerURL(input) || nip05.IsValidIdentifier(input) {
		bcsk := nostr.GeneratePrivateKey()
		oa := func(url string) { println("auth_url received but not handled") }

		if opts.BunkerClientSecretKey != "" {
			bcsk = opts.BunkerClientSecretKey
		}
		if opts.BunkerAuthHandler != nil {
			oa = opts.BunkerAuthHandler
		}

		bunker, err := nip46.ConnectBunker(ctx, bcsk, input, pool, oa)
		if err != nil {
			return nil, err
		}
		return BunkerSigner{bunker}, nil
	} else if prefix, parsed, err := nip19.Decode(input); err == nil && prefix == "nsec" {
		sec := parsed.(string)
		pk, _ := nostr.GetPublicKey(sec)
		return KeySigner{sec, pk, xsync.NewMapOf[string, [32]byte]()}, nil
	} else if _, err := hex.DecodeString(input); err == nil && len(input) <= 64 {
		input = strings.Repeat("0", 64-len(input)) + input // if the key is like '01', fill all the left zeroes
		pk, _ := nostr.GetPublicKey(input)
		return KeySigner{input, pk, xsync.NewMapOf[string, [32]byte]()}, nil
	}

	return nil, fmt.Errorf("unsupported input '%s'", input)
}
