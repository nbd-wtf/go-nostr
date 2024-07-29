package nostr

import (
	"encoding/hex"
	"net/url"
	"strings"
)

func IsValidRelayURL(u string) bool {
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}
	if parsed.Scheme != "wss" && parsed.Scheme != "ws" {
		return false
	}
	return true
}

func IsValid32ByteHex(thing string) bool {
	if strings.ToLower(thing) != thing {
		return false
	}
	if len(thing) != 64 {
		return false
	}
	_, err := hex.DecodeString(thing)
	return err == nil
}
