package nip05

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

type (
	name2KeyMap   map[string]string
	key2RelaysMap map[string][]string
)

type WellKnownResponse struct {
	Names  name2KeyMap   `json:"names"`  // NIP-05
	Relays key2RelaysMap `json:"relays"` // NIP-35
}

func QueryIdentifier(fullname string) *nostr.ProfilePointer {
	spl := strings.Split(fullname, "@")

	var name, domain string
	switch len(spl) {
	case 1:
		name = "_"
		domain = spl[0]
	case 2:
		name = spl[0]
		domain = spl[1]
	default:
		return nil
	}

	if strings.Index(domain, ".") == -1 {
		return nil
	}

	res, err := http.Get(fmt.Sprintf("https://%s/.well-known/nostr.json?name=%s", domain, name))
	if err != nil {
		return nil
	}

	var result WellKnownResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil
	}

	pubkey, _ := result.Names[name]
	relays, _ := result.Relays[pubkey]

	return &nostr.ProfilePointer{
		PublicKey: pubkey,
		Relays:    relays,
	}
}

func NormalizeIdentifier(fullname string) string {
	if strings.HasPrefix(fullname, "_@") {
		return fullname[2:]
	}

	return fullname
}
