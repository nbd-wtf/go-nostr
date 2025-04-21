package nip46

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/mailru/easyjson"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip04"
	"github.com/nbd-wtf/go-nostr/nip44"
)

var _ Signer = (*StaticKeySigner)(nil)

type StaticKeySigner struct {
	secretKey string

	sessionKeys []string
	sessions    []Session

	sync.Mutex

	RelaysToAdvertise map[string]RelayReadWrite
	AuthorizeRequest  func(harmless bool, from string, secret string) bool
}

func NewStaticKeySigner(secretKey string) StaticKeySigner {
	return StaticKeySigner{
		secretKey:         secretKey,
		RelaysToAdvertise: make(map[string]RelayReadWrite),
	}
}

func (p *StaticKeySigner) GetSession(clientPubkey string) (Session, bool) {
	idx, exists := slices.BinarySearch(p.sessionKeys, clientPubkey)
	if exists {
		return p.sessions[idx], true
	}
	return Session{}, false
}

func (p *StaticKeySigner) getOrCreateSession(clientPubkey string) (Session, error) {
	p.Lock()
	defer p.Unlock()

	idx, exists := slices.BinarySearch(p.sessionKeys, clientPubkey)
	if exists {
		return p.sessions[idx], nil
	}

	shared, err := nip04.ComputeSharedSecret(clientPubkey, p.secretKey)
	if err != nil {
		return Session{}, fmt.Errorf("failed to compute shared secret: %w", err)
	}

	ck, err := nip44.GenerateConversationKey(clientPubkey, p.secretKey)
	if err != nil {
		return Session{}, fmt.Errorf("failed to compute shared secret: %w", err)
	}

	pubkey, err := nostr.GetPublicKey(p.secretKey)
	if err != nil {
		return Session{}, fmt.Errorf("failed to derive public key: %w", err)
	}

	session := Session{
		PublicKey:       pubkey,
		SharedKey:       shared,
		ConversationKey: ck,
	}

	// add to pool
	p.sessionKeys = append(p.sessionKeys, "") // bogus append just to increase the capacity
	p.sessions = append(p.sessions, Session{})
	copy(p.sessionKeys[idx+1:], p.sessionKeys[idx:])
	copy(p.sessions[idx+1:], p.sessions[idx:])
	p.sessionKeys[idx] = clientPubkey
	p.sessions[idx] = session

	return session, nil
}

func (p *StaticKeySigner) HandleRequest(_ context.Context, event *nostr.Event) (
	req Request,
	resp Response,
	eventResponse nostr.Event,
	err error,
) {
	if event.Kind != nostr.KindNostrConnect {
		return req, resp, eventResponse,
			fmt.Errorf("event kind is %d, but we expected %d", event.Kind, nostr.KindNostrConnect)
	}

	session, err := p.getOrCreateSession(event.PubKey)
	if err != nil {
		return req, resp, eventResponse, err
	}

	req, err = session.ParseRequest(event)
	if err != nil {
		return req, resp, eventResponse, fmt.Errorf("error parsing request: %w", err)
	}

	var secret string
	var harmless bool
	var result string
	var resultErr error

	switch req.Method {
	case "connect":
		if len(req.Params) >= 2 {
			secret = req.Params[1]
		}
		result = "ack"
		harmless = true
	case "get_public_key":
		result = session.PublicKey
		harmless = true
	case "sign_event":
		if len(req.Params) != 1 {
			resultErr = fmt.Errorf("wrong number of arguments to 'sign_event'")
			break
		}
		evt := nostr.Event{}
		err = easyjson.Unmarshal([]byte(req.Params[0]), &evt)
		if err != nil {
			resultErr = fmt.Errorf("failed to decode event/2: %w", err)
			break
		}
		err = evt.Sign(p.secretKey)
		if err != nil {
			resultErr = fmt.Errorf("failed to sign event: %w", err)
			break
		}
		jrevt, _ := easyjson.Marshal(evt)
		result = string(jrevt)
	case "get_relays":
		jrelays, _ := json.Marshal(p.RelaysToAdvertise)
		result = string(jrelays)
		harmless = true
	case "nip44_encrypt":
		if len(req.Params) != 2 {
			resultErr = fmt.Errorf("wrong number of arguments to 'nip44_encrypt'")
			break
		}
		thirdPartyPubkey := req.Params[0]
		if !nostr.IsValidPublicKey(thirdPartyPubkey) {
			resultErr = fmt.Errorf("first argument to 'nip44_encrypt' is not a pubkey string")
			break
		}
		plaintext := req.Params[1]

		sharedSecret, err := nip44.GenerateConversationKey(thirdPartyPubkey, p.secretKey)
		if err != nil {
			resultErr = fmt.Errorf("failed to compute shared secret: %w", err)
			break
		}
		ciphertext, err := nip44.Encrypt(plaintext, sharedSecret)
		if err != nil {
			resultErr = fmt.Errorf("failed to encrypt: %w", err)
			break
		}
		result = ciphertext
	case "nip44_decrypt":
		if len(req.Params) != 2 {
			resultErr = fmt.Errorf("wrong number of arguments to 'nip44_decrypt'")
			break
		}
		thirdPartyPubkey := req.Params[0]
		if !nostr.IsValidPublicKey(thirdPartyPubkey) {
			resultErr = fmt.Errorf("first argument to 'nip44_decrypt' is not a pubkey string")
			break
		}
		ciphertext := req.Params[1]

		sharedSecret, err := nip44.GenerateConversationKey(thirdPartyPubkey, p.secretKey)
		if err != nil {
			resultErr = fmt.Errorf("failed to compute shared secret: %w", err)
			break
		}
		plaintext, err := nip44.Decrypt(ciphertext, sharedSecret)
		if err != nil {
			resultErr = fmt.Errorf("failed to encrypt: %w", err)
			break
		}
		result = plaintext
	case "nip04_encrypt":
		if len(req.Params) != 2 {
			resultErr = fmt.Errorf("wrong number of arguments to 'nip04_encrypt'")
			break
		}
		thirdPartyPubkey := req.Params[0]
		if !nostr.IsValidPublicKey(thirdPartyPubkey) {
			resultErr = fmt.Errorf("first argument to 'nip04_encrypt' is not a pubkey string")
			break
		}
		plaintext := req.Params[1]

		sharedSecret, err := nip04.ComputeSharedSecret(thirdPartyPubkey, p.secretKey)
		if err != nil {
			resultErr = fmt.Errorf("failed to compute shared secret: %w", err)
			break
		}
		ciphertext, err := nip04.Encrypt(plaintext, sharedSecret)
		if err != nil {
			resultErr = fmt.Errorf("failed to encrypt: %w", err)
			break
		}
		result = ciphertext
	case "nip04_decrypt":
		if len(req.Params) != 2 {
			resultErr = fmt.Errorf("wrong number of arguments to 'nip04_decrypt'")
			break
		}
		thirdPartyPubkey := req.Params[0]
		if !nostr.IsValidPublicKey(thirdPartyPubkey) {
			resultErr = fmt.Errorf("first argument to 'nip04_decrypt' is not a pubkey string")
			break
		}
		ciphertext := req.Params[1]

		sharedSecret, err := nip04.ComputeSharedSecret(thirdPartyPubkey, p.secretKey)
		if err != nil {
			resultErr = fmt.Errorf("failed to compute shared secret: %w", err)
			break
		}
		plaintext, err := nip04.Decrypt(ciphertext, sharedSecret)
		if err != nil {
			resultErr = fmt.Errorf("failed to encrypt: %w", err)
			break
		}
		result = plaintext
	case "ping":
		result = "pong"
		harmless = true
	default:
		return req, resp, eventResponse,
			fmt.Errorf("unknown method '%s'", req.Method)
	}

	if resultErr == nil && p.AuthorizeRequest != nil {
		if !p.AuthorizeRequest(harmless, event.PubKey, secret) {
			resultErr = fmt.Errorf("unauthorized")
		}
	}

	resp, eventResponse, err = session.MakeResponse(req.ID, event.PubKey, result, resultErr)
	if err != nil {
		return req, resp, eventResponse, err
	}

	err = eventResponse.Sign(p.secretKey)
	if err != nil {
		return req, resp, eventResponse, err
	}

	return req, resp, eventResponse, err
}
