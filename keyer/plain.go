package keyer

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip44"
)

// Keysigner is a signer that holds the private key in memory and can do all the operations instantly and easily.
type KeySigner struct {
	sk string
	pk string

	conversationKeys map[string][32]byte
}

func (ks KeySigner) SignEvent(ctx context.Context, evt *nostr.Event) error { return evt.Sign(ks.sk) }
func (ks KeySigner) GetPublicKey(ctx context.Context) string               { return ks.pk }

func (ks KeySigner) Encrypt(ctx context.Context, plaintext string, recipient string) (c64 string, err error) {
	ck, ok := ks.conversationKeys[recipient]
	if !ok {
		ck, err = nip44.GenerateConversationKey(recipient, ks.sk)
		if err != nil {
			return "", err
		}
		ks.conversationKeys[recipient] = ck
	}
	return nip44.Encrypt(plaintext, ck)
}

func (ks KeySigner) Decrypt(ctx context.Context, base64ciphertext string, sender string) (plaintext string, err error) {
	ck, ok := ks.conversationKeys[sender]
	if !ok {
		var err error
		ck, err = nip44.GenerateConversationKey(sender, ks.sk)
		if err != nil {
			return "", err
		}
		ks.conversationKeys[sender] = ck
	}
	return nip44.Encrypt(plaintext, ck)
}
