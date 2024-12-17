package nip46

import (
	"context"
	"net/url"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/nbd-wtf/go-nostr"
)

var json = jsoniter.ConfigFastest

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
	p, err := url.Parse(input)
	if err != nil {
		return false
	}
	if p.Scheme != "bunker" {
		return false
	}
	if !nostr.IsValidPublicKey(p.Host) {
		return false
	}
	if !strings.Contains(p.RawQuery, "relay=") {
		return false
	}
	return true
}
