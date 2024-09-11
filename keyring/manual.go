package keyring

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
)

// ManualSigner is a signer that doesn't really do anything, it just calls the functions given to it.
// It can be used when an app for some reason wants to ask the user to manually provide a signed event
// by copy-and-paste, for example.
type ManualSigner struct {
	ManualGetPublicKey func(context.Context) string
	ManualSignEvent    func(context.Context, *nostr.Event) error
	ManualEncrypt      func(ctx context.Context, plaintext string, recipientPublicKey string) (base64ciphertext string, err error)
	ManualDecrypt      func(ctx context.Context, base64ciphertext string, senderPublicKey string) (plaintext string, err error)
}

func (ms ManualSigner) SignEvent(ctx context.Context, evt *nostr.Event) error {
	return ms.ManualSignEvent(ctx, evt)
}

func (ms ManualSigner) GetPublicKey(ctx context.Context) string {
	return ms.ManualGetPublicKey(ctx)
}

func (ms ManualSigner) Encrypt(ctx context.Context, plaintext string, recipient string) (c64 string, err error) {
	return ms.ManualEncrypt(ctx, plaintext, recipient)
}

func (ms ManualSigner) Decrypt(ctx context.Context, base64ciphertext string, sender string) (plaintext string, err error) {
	return ms.ManualDecrypt(ctx, base64ciphertext, sender)
}
