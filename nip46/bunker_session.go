package nip46

import (
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip04"
	"github.com/nbd-wtf/go-nostr/nip44"
)

type Session struct {
	PublicKey       string
	SharedKey       []byte   // nip04
	ConversationKey [32]byte // nip44
}

type RelayReadWrite struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

func (s Session) ParseRequest(event *nostr.Event) (Request, error) {
	var req Request

	plain, err1 := nip44.Decrypt(event.Content, s.ConversationKey)
	if err1 != nil {
		var err2 error
		plain, err2 = nip04.Decrypt(event.Content, s.SharedKey)
		if err2 != nil {
			return req, fmt.Errorf("failed to decrypt event from %s: (nip44: %w, nip04: %w)", event.PubKey, err1, err2)
		}
	}

	err := json.Unmarshal([]byte(plain), &req)
	return req, err
}

func (s Session) MakeResponse(
	id string,
	requester string,
	result string,
	err error,
) (resp Response, evt nostr.Event, error error) {
	if err != nil {
		resp = Response{
			ID:    id,
			Error: err.Error(),
		}
	} else if result != "" {
		resp = Response{
			ID:     id,
			Result: result,
		}
	}

	jresp, _ := json.Marshal(resp)
	ciphertext, err := nip44.Encrypt(string(jresp), s.ConversationKey)
	if err != nil {
		return resp, evt, fmt.Errorf("failed to encrypt result: %w", err)
	}
	evt.Content = ciphertext
	evt.CreatedAt = nostr.Now()
	evt.Kind = nostr.KindNostrConnect
	evt.Tags = nostr.Tags{nostr.Tag{"p", requester}}

	return resp, evt, nil
}
