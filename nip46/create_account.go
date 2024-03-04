package nip46

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip04"
	"github.com/nbd-wtf/go-nostr/nip05"
)

func CheckNameAvailability(ctx context.Context, name, domain string) bool {
	result, _, err := nip05.Fetch(ctx, name+"@"+domain)
	if err != nil {
		return false
	}
	_, ok := result.Names[name]
	return !ok
}

type CreateAccountOptions struct {
	Email string
}

func CreateAccount(
	ctx context.Context,
	clientSecretKey string,
	name string,
	domain string,
	pool *nostr.SimplePool,
	extraOpts *CreateAccountOptions,
	onAuth func(string),
) (*BunkerClient, error) {
	if pool == nil {
		pool = nostr.NewSimplePool(ctx)
	}

	// create a bunker that targets the provider directly
	providerPubkey, relays, err := queryWellKnownNostrJson(ctx, domain)
	if err != nil {
		return nil, err
	}

	bunker := NewBunker(
		ctx,
		clientSecretKey,
		providerPubkey,
		relays,
		pool,
		onAuth,
	)

	_, err = bunker.RPC(ctx, "connect", []string{providerPubkey, ""})
	if err != nil {
		return nil, fmt.Errorf("initial connect error: %w", err)
	}

	// call create_account on it, it should return the value of the public key that will be created
	email := ""
	if extraOpts != nil {
		email = extraOpts.Email
	}
	resp, err := bunker.RPC(ctx, "create_account", []string{name, domain, email})
	if err != nil {
		return nil, fmt.Errorf("error on create_account: %w", err)
	}

	newlyCreatedPublicKey := resp

	// update this bunker instance so it targets the new key now instead of the provider
	bunker.target = newlyCreatedPublicKey
	bunker.sharedSecret, _ = nip04.ComputeSharedSecret(newlyCreatedPublicKey, clientSecretKey)
	bunker.getPublicKeyResponse = newlyCreatedPublicKey

	// finally try to connect again using the new key as the target
	_, err = bunker.RPC(ctx, "connect", []string{newlyCreatedPublicKey, ""})
	if err != nil {
		return nil, fmt.Errorf("newly-created public key connect error: %w", err)
	}

	return bunker, err
}
