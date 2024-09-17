package keyer

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip44"
	"github.com/puzpuzpuz/xsync/v3"
)

// Keysigner is a signer that holds the private key in memory and can do all the operations instantly and easily.
type KeySigner struct {
	sk string
	pk string

	conversationKeys *xsync.MapOf[string, [32]byte]
}

func NewPlainKeySigner(sec string) KeySigner {
	pk, _ := nostr.GetPublicKey(sec)
	return KeySigner{sec, pk, xsync.NewMapOf[string, [32]byte]()}
}

func (ks KeySigner) SignEvent(ctx context.Context, evt *nostr.Event) error { return evt.Sign(ks.sk) }
func (ks KeySigner) GetPublicKey(ctx context.Context) string               { return ks.pk }

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
