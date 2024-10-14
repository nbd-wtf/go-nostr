package nip46

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip04"
	"github.com/nbd-wtf/go-nostr/nip44"
)

var BUNKER_REGEX = regexp.MustCompile(`^bunker:\/\/([0-9a-f]{64})\??([?\/\w:.=&%]*)$`)

type Request struct {
	ID     string   `json:"id"`
	Method string   `json:"method"`
	Params []string `json:"params"`
}

func (r Request) String() string {
	j, _ := json.Marshal(r)
	return string(j)
}

type Response struct {
	ID     string `json:"id"`
	Error  string `json:"error,omitempty"`
	Result string `json:"result,omitempty"`
}

func (r Response) String() string {
	j, _ := json.Marshal(r)
	return string(j)
}

type Signer interface {
	GetSession(clientPubkey string) (Session, bool)
	HandleRequest(context.Context, *nostr.Event) (req Request, resp Response, eventResponse nostr.Event, err error)
}

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

	plain, err := nip44.Decrypt(event.Content, s.ConversationKey)
	if err != nil {
		plain, err = nip04.Decrypt(event.Content, s.SharedKey)
		if err != nil {
			return req, fmt.Errorf("failed to decrypt event from %s: %w", event.PubKey, err)
		}
	}

	err = json.Unmarshal([]byte(plain), &req)
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
	ciphertext, err := nip04.Encrypt(string(jresp), s.SharedKey)
	if err != nil {
		return resp, evt, fmt.Errorf("failed to encrypt result: %w", err)
	}
	evt.Content = ciphertext
	evt.CreatedAt = nostr.Now()
	evt.Kind = nostr.KindNostrConnect
	evt.Tags = nostr.Tags{nostr.Tag{"p", requester}}

	return resp, evt, nil
}

func IsValidBunkerURL(input string) bool {
	return BUNKER_REGEX.MatchString(input)
}
