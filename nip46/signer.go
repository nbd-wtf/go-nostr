package nip46

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/mailru/easyjson"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip04"
	"golang.org/x/exp/slices"
)

type Request struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	Params []any  `json:"params"`
}

type Response struct {
	ID     string `json:"id"`
	Error  string `json:"error,omitempty"`
	Result any    `json:"result,omitempty"`
}

type Session struct {
	SharedKey []byte
}

func (s Session) ParseRequest(event *nostr.Event) (Request, error) {
	var req Request

	plain, err := nip04.Decrypt(event.Content, s.SharedKey)
	if err != nil {
		return req, fmt.Errorf("failed to decrypt event from %s: %w", event.PubKey, err)
	}

	err = json.Unmarshal([]byte(plain), &req)
	return req, err
}

func (s Session) MakeResponse(
	id string,
	requester string,
	result any,
	err error,
) (resp Response, evt nostr.Event, error error) {
	if err != nil {
		resp = Response{
			ID:    id,
			Error: err.Error(),
		}
	} else if result != nil {
		resp = Response{
			ID:     id,
			Result: result,
		}
	}

	jresp, _ := json.Marshal(resp)
	ciphertext, err := nip04.Encrypt(string(jresp), s.SharedKey)
	if err != nil {
		return resp, evt, fmt.Errorf("failed to encrypt result: %w", err)
	}
	evt.Content = ciphertext

	evt.CreatedAt = nostr.Now()
	evt.Kind = nostr.KindNostrConnect
	evt.Tags = nostr.Tags{nostr.Tag{"p", requester}}

	err = evt.Sign(hex.EncodeToString(s.SharedKey))
	if err != nil {
		return resp, evt, err
	}
	return resp, evt, nil
}

type Pool struct {
	secretKey string

	sessionKeys []string
	sessions    []Session

	RelaysToAdvertise map[string]relayReadWrite
}

type relayReadWrite struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

func NewPool(secretKey string) Pool {
	return Pool{secretKey: secretKey}
}

func (p *Pool) AddRelay(url string, read bool, write bool) {
	p.RelaysToAdvertise[url] = relayReadWrite{read, write}
}

func (p *Pool) GetSession(clientPubkey string) (Session, error) {
	idx, exists := slices.BinarySearch(p.sessionKeys, clientPubkey)
	if exists {
		return p.sessions[idx], nil
	}

	shared, err := nip04.ComputeSharedSecret(clientPubkey, p.secretKey)
	if err != nil {
		return Session{}, fmt.Errorf("failed to compute shared secret: %w", err)
	}

	session := Session{
		SharedKey: shared,
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

func (p *Pool) HandleRequest(event *nostr.Event) (req Request, resp Response, eventResponse nostr.Event, harmless bool, err error) {
	if event.Kind != nostr.KindNostrConnect {
		return req, resp, eventResponse, false,
			fmt.Errorf("event kind is %d, but we expected %d", event.Kind, nostr.KindNostrConnect)
	}

	session, err := p.GetSession(event.PubKey)
	if err != nil {
		return req, resp, eventResponse, false, err
	}

	req, err = session.ParseRequest(event)
	if err != nil {
		return req, resp, eventResponse, false, fmt.Errorf("error parsing request: %w", err)
	}

	var result any
	var resultErr error

	switch req.Method {
	case "connect":
		result = map[string]any{}
		harmless = true
	case "get_public_key":
		pubkey, err := nostr.GetPublicKey(p.secretKey)
		if err != nil {
			resultErr = fmt.Errorf("failed to derive public key: %w", err)
			break
		} else {
			result = pubkey
			harmless = true
		}
	case "sign_event":
		if len(req.Params) != 1 {
			resultErr = fmt.Errorf("wrong number of arguments to 'sign_event'")
			break
		}
		jevt, err := json.Marshal(req.Params[0])
		if err != nil {
			resultErr = fmt.Errorf("failed to decode event/1: %w", err)
			break
		}
		evt := nostr.Event{}
		err = easyjson.Unmarshal(jevt, &evt)
		if err != nil {
			resultErr = fmt.Errorf("failed to decode event/2: %w", err)
			break
		}
		err = evt.Sign(p.secretKey)
		if err != nil {
			resultErr = fmt.Errorf("failed to sign event: %w", err)
			break
		}
		result = evt
	case "get_relays":
		result = p.RelaysToAdvertise
		harmless = true
	case "nip04_encrypt":
		if len(req.Params) != 2 {
			resultErr = fmt.Errorf("wrong number of arguments to 'nip04_encrypt'")
			break
		}
		thirdPartyPubkey, ok := req.Params[0].(string)
		if !ok || !nostr.IsValidPublicKeyHex(thirdPartyPubkey) {
			resultErr = fmt.Errorf("first argument to 'nip04_encrypt' is not a pubkey string")
			break
		}
		plaintext, ok := req.Params[1].(string)
		if !ok {
			resultErr = fmt.Errorf("second argument to 'nip04_encrypt' is not a string")
			break
		}
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
		thirdPartyPubkey, ok := req.Params[0].(string)
		if !ok || !nostr.IsValidPublicKeyHex(thirdPartyPubkey) {
			resultErr = fmt.Errorf("first argument to 'nip04_decrypt' is not a pubkey string")
			break
		}
		ciphertext, ok := req.Params[1].(string)
		if !ok {
			resultErr = fmt.Errorf("second argument to 'nip04_decrypt' is not a string")
			break
		}
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
	default:
		return req, resp, eventResponse, false,
			fmt.Errorf("unknown method '%s'", req.Method)
	}

	resp, eventResponse, err = session.MakeResponse(req.ID, event.PubKey, result, resultErr)
	return req, resp, eventResponse, harmless, err
}
