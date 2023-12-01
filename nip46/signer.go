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

func (s Session) MakeResultResponse(id string, result any) (nostr.Event, error) {
	evt := nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindNostrConnect,
		Tags:      nostr.Tags{},
	}

	data, err := json.Marshal(result)
	if err != nil {
		return evt, fmt.Errorf("failed to encode result to json: %w", err)
	}
	ciphertext, err := nip04.Encrypt(string(data), s.SharedKey)
	if err != nil {
		return evt, fmt.Errorf("failed to encrypt result: %w", err)
	}
	evt.Content = ciphertext

	err = evt.Sign(hex.EncodeToString(s.SharedKey))
	if err != nil {
		return evt, err
	}
	return evt, nil
}

func (s Session) MakeErrorResponse(id string, err error) (nostr.Event, error) {
	evt := nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindNostrConnect,
		Tags:      nostr.Tags{},
	}

	resp, _ := json.Marshal(Response{
		ID:    id,
		Error: err.Error(),
	})

	ciphertext, err := nip04.Encrypt(string(resp), s.SharedKey)
	if err != nil {
		return evt, fmt.Errorf("failed to encrypt result: %w", err)
	}
	evt.Content = ciphertext

	err = evt.Sign(hex.EncodeToString(s.SharedKey))
	if err != nil {
		return evt, err
	}
	return evt, nil
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

func (p *Pool) HandleRequest(event *nostr.Event) (req Request, resp nostr.Event, err error) {
	if event.Kind != nostr.KindNostrConnect {
		return req, resp, fmt.Errorf("event kind is %d, but we expected %d",
			event.Kind, nostr.KindNostrConnect)
	}

	session, err := p.GetSession(event.PubKey)
	if err != nil {
		return req, resp, err
	}

	req, err = session.ParseRequest(event)
	if err != nil {
		return req, resp, fmt.Errorf("error parsing request: %w", err)
	}

	var result any
	var resultErr error

	switch req.Method {
	case "connect":
		result = map[string]any{}
	case "get_public_key":
		pubkey, err := nostr.GetPublicKey(p.secretKey)
		if err != nil {
			resultErr = fmt.Errorf("failed to derive public key: %w", err)
			goto end
		} else {
			result = pubkey
		}
	case "sign_event":
		if len(req.Params) != 1 {
			resultErr = fmt.Errorf("wrong number of arguments to 'sign_event'")
			goto end
		}
		jevt, err := json.Marshal(req.Params[0])
		if err != nil {
			resultErr = fmt.Errorf("failed to decode event/1: %w", err)
			goto end
		}
		evt := nostr.Event{}
		err = easyjson.Unmarshal(jevt, &evt)
		if err != nil {
			resultErr = fmt.Errorf("failed to decode event/2: %w", err)
			goto end
		}
		err = evt.Sign(p.secretKey)
		if err != nil {
			resultErr = fmt.Errorf("failed to sign event: %w", err)
			goto end
		}
		result = evt
	case "get_relays":
		result = p.RelaysToAdvertise
	case "nip04_encrypt":
		if len(req.Params) != 2 {
			resultErr = fmt.Errorf("wrong number of arguments to 'nip04_encrypt'")
			goto end
		}
		thirdPartyPubkey, ok := req.Params[0].(string)
		if !ok || !nostr.IsValidPublicKeyHex(thirdPartyPubkey) {
			resultErr = fmt.Errorf("first argument to 'nip04_encrypt' is not a pubkey string")
			goto end
		}
		plaintext, ok := req.Params[1].(string)
		if !ok {
			resultErr = fmt.Errorf("second argument to 'nip04_encrypt' is not a string")
			goto end
		}
		sharedSecret, err := nip04.ComputeSharedSecret(thirdPartyPubkey, p.secretKey)
		if err != nil {
			resultErr = fmt.Errorf("failed to compute shared secret: %w", err)
			goto end
		}
		ciphertext, err := nip04.Encrypt(plaintext, sharedSecret)
		if err != nil {
			resultErr = fmt.Errorf("failed to encrypt: %w", err)
			goto end
		}
		result = ciphertext
	case "nip04_decrypt":
		if len(req.Params) != 2 {
			resultErr = fmt.Errorf("wrong number of arguments to 'nip04_decrypt'")
			goto end
		}
		thirdPartyPubkey, ok := req.Params[0].(string)
		if !ok || !nostr.IsValidPublicKeyHex(thirdPartyPubkey) {
			resultErr = fmt.Errorf("first argument to 'nip04_decrypt' is not a pubkey string")
			goto end
		}
		ciphertext, ok := req.Params[1].(string)
		if !ok {
			resultErr = fmt.Errorf("second argument to 'nip04_decrypt' is not a string")
			goto end
		}
		sharedSecret, err := nip04.ComputeSharedSecret(thirdPartyPubkey, p.secretKey)
		if err != nil {
			resultErr = fmt.Errorf("failed to compute shared secret: %w", err)
			goto end
		}
		plaintext, err := nip04.Decrypt(ciphertext, sharedSecret)
		if err != nil {
			resultErr = fmt.Errorf("failed to encrypt: %w", err)
			goto end
		}
		result = plaintext
	default:
		return req, resp, fmt.Errorf("unknown method '%s'", req.Method)
	}

end:
	if resultErr != nil {
		resp, err = session.MakeErrorResponse(req.ID, resultErr)
	} else if result != nil {
		resp, err = session.MakeResultResponse(req.ID, map[string]any{})
	}
	if err != nil {
		return req, resp, fmt.Errorf("failed to encrypt '%s' result", req.Method)
	}
	return req, resp, nil
}
