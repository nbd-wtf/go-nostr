package nip46

import (
	"context"
	"regexp"

	jsoniter "github.com/json-iterator/go"
	"github.com/nbd-wtf/go-nostr"
)

var (
	BUNKER_REGEX = regexp.MustCompile(`^bunker:\/\/([0-9a-f]{64})\??([?\/\w:.=&%]*)$`)
	json         = jsoniter.ConfigFastest
)

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

func IsValidBunkerURL(input string) bool {
	return BUNKER_REGEX.MatchString(input)
}
