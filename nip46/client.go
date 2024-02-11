package nip46

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"sync/atomic"

	"github.com/mailru/easyjson"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip04"
	"github.com/puzpuzpuz/xsync/v3"
)

type BunkerClient struct {
	serial          atomic.Uint64
	clientSecretKey string
	pool            *nostr.SimplePool
	target          string
	relays          []string
	sharedSecret    []byte
	listeners       *xsync.MapOf[string, chan Response]
	idPrefix        string

	// memoized
	getPublicKeyResponse string
}

// ConnectBunker establishes an RPC connection to a NIP-46 signer using the relays and secret provided in the bunkerURL.
// pool can be passed to reuse an existing pool, otherwise a new pool will be created.
func ConnectBunker(
	ctx context.Context,
	clientSecretKey string,
	bunkerURL string,
	pool *nostr.SimplePool,
) (*BunkerClient, error) {
	parsed, err := url.Parse(bunkerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	if parsed.Scheme != "bunker" {
		return nil, fmt.Errorf("wrong scheme '%s', must be bunker://", parsed.Scheme)
	}

	target := parsed.Host
	if !nostr.IsValidPublicKey(target) {
		return nil, fmt.Errorf("'%s' is not a valid public key hex", target)
	}

	secret := parsed.Query().Get("secret")
	relays := parsed.Query()["relay"]

	if pool == nil {
		pool = nostr.NewSimplePool(ctx)
	}

	shared, err := nip04.ComputeSharedSecret(target, clientSecretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to compute shared secret: %w", err)
	}

	clientPubKey, _ := nostr.GetPublicKey(clientSecretKey)
	bunker := &BunkerClient{
		clientSecretKey: clientSecretKey,
		pool:            pool,
		target:          target,
		relays:          relays,
		sharedSecret:    shared,
		listeners:       xsync.NewMapOf[string, chan Response](),
		idPrefix:        "gn-" + strconv.Itoa(rand.Intn(65536)),
	}

	go func() {
		events := pool.SubMany(ctx, relays, nostr.Filters{
			{
				Tags:  nostr.TagMap{"p": []string{clientPubKey}},
				Kinds: []int{nostr.KindNostrConnect},
			},
		})
		for ie := range events {
			if ie.Kind != nostr.KindNostrConnect {
				continue
			}

			var resp Response
			plain, err := nip04.Decrypt(ie.Content, shared)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(plain), &resp)
			if err != nil {
				continue
			}

			if dispatcher, ok := bunker.listeners.Load(resp.ID); ok {
				dispatcher <- resp
			}
		}
	}()

	ourPubkey, _ := nostr.GetPublicKey(clientSecretKey)
	_, err = bunker.RPC(ctx, "connect", []string{ourPubkey, secret})
	return bunker, err
}

func (bunker *BunkerClient) Ping(ctx context.Context) error {
	_, err := bunker.RPC(ctx, "ping", []string{})
	if err != nil {
		return err
	}
	return nil
}

func (bunker *BunkerClient) GetPublicKey(ctx context.Context) (string, error) {
	if bunker.getPublicKeyResponse != "" {
		return bunker.getPublicKeyResponse, nil
	}
	resp, err := bunker.RPC(ctx, "get_public_key", []string{})
	bunker.getPublicKeyResponse = resp
	return resp, err
}

func (bunker *BunkerClient) SignEvent(ctx context.Context, evt *nostr.Event) error {
	resp, err := bunker.RPC(ctx, "sign_event", []string{evt.String()})
	if err == nil {
		err = easyjson.Unmarshal([]byte(resp), evt)
	}
	return err
}

func (bunker *BunkerClient) RPC(ctx context.Context, method string, params []string) (string, error) {
	id := bunker.idPrefix + "-" + strconv.FormatUint(bunker.serial.Add(1), 10)
	req, err := json.Marshal(Request{
		ID:     id,
		Method: method,
		Params: params,
	})
	if err != nil {
		return "", err
	}

	content, err := nip04.Encrypt(string(req), bunker.sharedSecret)
	if err != nil {
		return "", fmt.Errorf("error encrypting request: %w", err)
	}

	evt := nostr.Event{
		Content:   content,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindNostrConnect,
		Tags:      nostr.Tags{{"p", bunker.target}},
	}
	if err := evt.Sign(bunker.clientSecretKey); err != nil {
		return "", fmt.Errorf("failed to sign request event: %w", err)
	}

	respWaiter := make(chan Response)
	bunker.listeners.Store(id, respWaiter)

	hasWorked := false
	for _, r := range bunker.relays {
		relay, err := bunker.pool.EnsureRelay(r)
		if err == nil {
			hasWorked = true
		}
		relay.Publish(ctx, evt)
	}
	if !hasWorked {
		return "", fmt.Errorf("couldn't connect to any relay")
	}

	resp := <-respWaiter
	if resp.Error != "" {
		return "", fmt.Errorf("response error: %s", resp.Error)
	}

	return resp.Result, nil
}
