package keyer

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip44"
	"github.com/nbd-wtf/go-nostr/nip49"
)

// EncryptedKeySigner is a signer that must always ask the user for a password before every operation.
type EncryptedKeySigner struct {
	ncryptsec string
	pk        string
	callback  func(context.Context) string
}

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

func (es *EncryptedKeySigner) SignEvent(ctx context.Context, evt *nostr.Event) error {
	password := es.callback(ctx)
	sk, err := nip49.Decrypt(es.ncryptsec, password)
	if err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}
	es.pk = evt.PubKey
	return evt.Sign(sk)
}

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
