package nip05

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func QueryIdentifier(fullname string) string {
	spl := strings.Split(fullname, "@")
	if len(spl) != 2 {
		return ""
	}

	name := spl[0]
	domain := spl[1]
	res, err := http.Get(fmt.Sprintf("https://%s/.well-known/nostr.json?name=%s", domain, name))
	if err != nil {
		return ""
	}

	var result struct {
		Names map[string]string `json:"names"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return ""
	}

	pubkey, _ := result.Names[name]
	return pubkey
}

func NormalizeIdentifier(fullname string) string {
	if strings.HasPrefix(fullname, "_@") {
		return fullname[2:]
	}

	return fullname
}
