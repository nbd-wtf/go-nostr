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

var _ Signer = (*DynamicSigner)(nil)

type DynamicSigner struct {
	sessionKeys []string
	sessions    []Session

	sync.Mutex

	getHandlerSecretKey func(handlerPubkey string) (string, error)
	getUserKeyer        func(handlerPubkey string) (nostr.Keyer, error)
	authorizeSigning    func(event nostr.Event, from string, secret string) bool
	authorizeEncryption func(from string, secret string) bool
	onEventSigned       func(event nostr.Event)
	getRelays           func(pubkey string) map[string]RelayReadWrite
}

func NewDynamicSigner(
	// the handler is the keypair we use to communicate with the NIP-46 client, decrypt requests, encrypt responses etc
	getHandlerSecretKey func(handlerPubkey string) (string, error),

	// this should correspond to the actual user on behalf of which we will respond to requests
	getUserKeyer func(handlerPubkey string) (nostr.Keyer, error),

	// this is called on every sign_event call, if it is nil it will be assumed that everything is authorized
	authorizeSigning func(event nostr.Event, from string, secret string) bool,

	// this is called on every encrypt or decrypt calls, if it is nil it will be assumed that everything is authorized
	authorizeEncryption func(from string, secret string) bool,

	// unless it is nil, this is called after every event is signed
	onEventSigned func(event nostr.Event),

	// unless it is nil, the results of this will be used in reply to get_relays
	getRelays func(pubkey string) map[string]RelayReadWrite,
) DynamicSigner {
	return DynamicSigner{
		getHandlerSecretKey: getHandlerSecretKey,
		getUserKeyer:        getUserKeyer,
		authorizeSigning:    authorizeSigning,
		authorizeEncryption: authorizeEncryption,
		onEventSigned:       onEventSigned,
		getRelays:           getRelays,
	}
}

func (p *DynamicSigner) GetSession(clientPubkey string) (Session, bool) {
	idx, exists := slices.BinarySearch(p.sessionKeys, clientPubkey)
	if exists {
		return p.sessions[idx], true
	}
	return Session{}, false
}

func (p *DynamicSigner) setSession(clientPubkey string, session Session) {
	p.Lock()
	defer p.Unlock()

	idx, exists := slices.BinarySearch(p.sessionKeys, clientPubkey)
	if exists {
		return
	}

	// add to pool
	p.sessionKeys = append(p.sessionKeys, "") // bogus append just to increase the capacity
	p.sessions = append(p.sessions, Session{})
	copy(p.sessionKeys[idx+1:], p.sessionKeys[idx:])
	copy(p.sessions[idx+1:], p.sessions[idx:])
	p.sessionKeys[idx] = clientPubkey
	p.sessions[idx] = session
}

func (p *DynamicSigner) HandleRequest(ctx context.Context, event *nostr.Event) (
	req Request,
	resp Response,
	eventResponse nostr.Event,
	err error,
) {
	if event.Kind != nostr.KindNostrConnect {
		return req, resp, eventResponse,
			fmt.Errorf("event kind is %d, but we expected %d", event.Kind, nostr.KindNostrConnect)
	}

	handler := event.Tags.Find("p")
	if handler == nil || !nostr.IsValid32ByteHex(handler[1]) {
		return req, resp, eventResponse, fmt.Errorf("invalid \"p\" tag")
	}

	handlerPubkey := handler[1]
	handlerSecret, err := p.getHandlerSecretKey(handlerPubkey)
	if err != nil {
		return req, resp, eventResponse, fmt.Errorf("no private key for %s: %w", handlerPubkey, err)
	}
	userKeyer, err := p.getUserKeyer(handlerPubkey)
	if err != nil {
		return req, resp, eventResponse, fmt.Errorf("failed to get user keyer for %s: %w", handlerPubkey, err)
	}

	var session Session
	idx, exists := slices.BinarySearch(p.sessionKeys, event.PubKey)
	if exists {
		session = p.sessions[idx]
	} else {
		session = Session{}

		session.SharedKey, err = nip04.ComputeSharedSecret(event.PubKey, handlerSecret)
		if err != nil {
			return req, resp, eventResponse, fmt.Errorf("failed to compute shared secret: %w", err)
		}

		session.ConversationKey, err = nip44.GenerateConversationKey(event.PubKey, handlerSecret)
		if err != nil {
			return req, resp, eventResponse, fmt.Errorf("failed to compute shared secret: %w", err)
		}

		session.PublicKey, err = userKeyer.GetPublicKey(ctx)
		if err != nil {
			return req, resp, eventResponse, fmt.Errorf("failed to get public key: %w", err)
		}

		p.setSession(event.PubKey, session)
	}

	req, err = session.ParseRequest(event)
	if err != nil {
		return req, resp, eventResponse, fmt.Errorf("error parsing request: %w", err)
	}

	var secret string
	var result string
	var resultErr error

	switch req.Method {
	case "connect":
		if len(req.Params) >= 2 {
			secret = req.Params[1]
		}
		result = "ack"
	case "get_public_key":
		result = session.PublicKey
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
		if p.authorizeSigning != nil && !p.authorizeSigning(evt, event.PubKey, secret) {
			resultErr = fmt.Errorf("refusing to sign this event")
			break
		}

		err = userKeyer.SignEvent(ctx, &evt)
		if err != nil {
			resultErr = fmt.Errorf("failed to sign event: %w", err)
			break
		}
		jrevt, _ := easyjson.Marshal(evt)
		result = string(jrevt)
	case "get_relays":
		if p.getRelays == nil {
			jrelays, _ := json.Marshal(p.getRelays(session.PublicKey))
			result = string(jrelays)
		} else {
			result = "{}"
		}
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
		if p.authorizeEncryption != nil && !p.authorizeEncryption(event.PubKey, secret) {
			resultErr = fmt.Errorf("refusing to encrypt")
			break
		}
		plaintext := req.Params[1]

		ciphertext, err := userKeyer.Encrypt(ctx, plaintext, thirdPartyPubkey)
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
		if p.authorizeEncryption != nil && !p.authorizeEncryption(event.PubKey, secret) {
			resultErr = fmt.Errorf("refusing to decrypt")
			break
		}
		ciphertext := req.Params[1]

		plaintext, err := userKeyer.Decrypt(ctx, ciphertext, thirdPartyPubkey)
		if err != nil {
			resultErr = fmt.Errorf("failed to decrypt: %w", err)
			break
		}
		result = plaintext
	case "ping":
		result = "pong"
	default:
		return req, resp, eventResponse,
			fmt.Errorf("unknown method '%s'", req.Method)
	}

	resp, eventResponse, err = session.MakeResponse(req.ID, event.PubKey, result, resultErr)
	if err != nil {
		return req, resp, eventResponse, err
	}

	err = eventResponse.Sign(handlerSecret)
	if err != nil {
		return req, resp, eventResponse, err
	}

	return req, resp, eventResponse, err
}
