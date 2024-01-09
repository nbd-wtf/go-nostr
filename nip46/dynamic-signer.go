package nip46

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/mailru/easyjson"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip04"
	"golang.org/x/exp/slices"
)

var _ Signer = (*DynamicSigner)(nil)

type DynamicSigner struct {
	sessionKeys []string
	sessions    []Session

	sync.Mutex

	RelaysToAdvertise map[string]RelayReadWrite

	getPrivateKey    func(pubkey string) (string, error)
	authorizeSigning func(event nostr.Event) bool
	onEventSigned    func(event nostr.Event)
	authorizeNIP04   func() bool
}

func NewDynamicSigner(
	getPrivateKey func(pubkey string) (string, error),
	authorizeSigning func(event nostr.Event) bool,
	onEventSigned func(event nostr.Event),
	authorizeNIP04 func() bool,
) DynamicSigner {
	return DynamicSigner{
		getPrivateKey:     getPrivateKey,
		authorizeSigning:  authorizeSigning,
		onEventSigned:     onEventSigned,
		authorizeNIP04:    authorizeNIP04,
		RelaysToAdvertise: make(map[string]RelayReadWrite),
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

func (p *DynamicSigner) HandleRequest(event *nostr.Event) (
	req Request,
	resp Response,
	eventResponse nostr.Event,
	harmless bool,
	err error,
) {
	if event.Kind != nostr.KindNostrConnect {
		return req, resp, eventResponse, false,
			fmt.Errorf("event kind is %d, but we expected %d", event.Kind, nostr.KindNostrConnect)
	}

	targetUser := event.Tags.GetFirst([]string{"p", ""})
	if targetUser == nil || !nostr.IsValidPublicKeyHex((*targetUser)[1]) {
		return req, resp, eventResponse, false, fmt.Errorf("invalid \"p\" tag")
	}

	targetPubkey := (*targetUser)[1]

	privateKey, err := p.getPrivateKey(targetPubkey)
	if err != nil {
		return req, resp, eventResponse, false, fmt.Errorf("no private key for %s: %w", targetPubkey, err)
	}

	var session Session
	idx, exists := slices.BinarySearch(p.sessionKeys, event.PubKey)
	if exists {
		session = p.sessions[idx]
	} else {
		session = Session{}

		session.SharedKey, err = nip04.ComputeSharedSecret(event.PubKey, privateKey)
		if err != nil {
			return req, resp, eventResponse, false, fmt.Errorf("failed to compute shared secret: %w", err)
		}
		p.setSession(event.PubKey, session)

		req, err = session.ParseRequest(event)
		if err != nil {
			return req, resp, eventResponse, false, fmt.Errorf("error parsing request: %w", err)
		}
	}

	var result string
	var resultErr error

	switch req.Method {
	case "connect":
		result = "ack"
		harmless = true
	case "get_public_key":
		result = targetPubkey
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
		if !p.authorizeSigning(evt) {
			resultErr = fmt.Errorf("refusing to sign this event")
			break
		}
		err = evt.Sign(privateKey)
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
	case "nip04_encrypt":
		if len(req.Params) != 2 {
			resultErr = fmt.Errorf("wrong number of arguments to 'nip04_encrypt'")
			break
		}
		thirdPartyPubkey := req.Params[0]
		if !nostr.IsValidPublicKeyHex(thirdPartyPubkey) {
			resultErr = fmt.Errorf("first argument to 'nip04_encrypt' is not a pubkey string")
			break
		}
		if !p.authorizeNIP04() {
			resultErr = fmt.Errorf("refusing to encrypt")
			break
		}
		plaintext := req.Params[1]
		sharedSecret, err := nip04.ComputeSharedSecret(thirdPartyPubkey, privateKey)
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
		if !nostr.IsValidPublicKeyHex(thirdPartyPubkey) {
			resultErr = fmt.Errorf("first argument to 'nip04_decrypt' is not a pubkey string")
			break
		}
		if !p.authorizeNIP04() {
			resultErr = fmt.Errorf("refusing to decrypt")
			break
		}
		ciphertext := req.Params[1]
		sharedSecret, err := nip04.ComputeSharedSecret(thirdPartyPubkey, privateKey)
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
	default:
		return req, resp, eventResponse, false,
			fmt.Errorf("unknown method '%s'", req.Method)
	}

	resp, eventResponse, err = session.MakeResponse(req.ID, event.PubKey, result, resultErr)
	if err != nil {
		return req, resp, eventResponse, harmless, err
	}

	err = eventResponse.Sign(privateKey)
	if err != nil {
		return req, resp, eventResponse, harmless, err
	}

	return req, resp, eventResponse, harmless, err
}
