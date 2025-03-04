package keyer

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

// ReadOnlySigner is a Signer that holds a public key in memory and cannot sign anything
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
