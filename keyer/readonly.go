package keyer

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

var (
	_ nostr.User   = (*ReadOnlyUser)(nil)
	_ nostr.Signer = (*ReadOnlySigner)(nil)
)

// ReadOnlyUser is a nostr.User that has this public key
type ReadOnlyUser struct {
	pk string
}

func NewReadOnlyUser(pk string) ReadOnlyUser {
	return ReadOnlyUser{pk}
}

// GetPublicKey returns the public key associated with this signer.
func (ros ReadOnlyUser) GetPublicKey(context.Context) (string, error) {
	return ros.pk, nil
}

// ReadOnlySigner is like a ReadOnlyUser, but has a fake GetPublicKey method that doesn't work.
type ReadOnlySigner struct {
	pk string
}

func NewReadOnlySigner(pk string) ReadOnlySigner {
	return ReadOnlySigner{pk}
}

// SignEvent returns an error.
func (ros ReadOnlySigner) SignEvent(context.Context, *nostr.Event) error {
	return fmt.Errorf("read-only, we don't have the secret key, cannot sign")
}

// GetPublicKey returns the public key associated with this signer.
func (ros ReadOnlySigner) GetPublicKey(context.Context) (string, error) {
	return ros.pk, nil
}
