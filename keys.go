package nostr

import (
	"encoding/hex"

	"github.com/fiatjaf/bip340"
)

func GeneratePrivateKey() string {
	return hex.EncodeToString(bip340.GeneratePrivateKey().Bytes())
}

func GetPublicKey(sk string) (string, error) {
	privateKey, err := bip340.ParsePrivateKey(sk)
	if err != nil {
		return "", err
	}

	publicKey := bip340.GetPublicKey(privateKey)
	return hex.EncodeToString(publicKey[:]), nil
}
