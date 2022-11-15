package nip19

import (
	"encoding/hex"
	"strings"
)

// TranslatePublicKey turns a hex or bech32 public key into always hex
func TranslatePublicKey(bech32orHexKey string) string {
	if strings.HasPrefix(bech32orHexKey, "npub1") {
		data, _, _ := Decode(bech32orHexKey)
		return hex.EncodeToString(data)
	}

	// just return what we got
	return bech32orHexKey
}
