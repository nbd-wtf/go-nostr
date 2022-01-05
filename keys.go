package nostr

import (
	"crypto/ecdsa"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/tyler-smith/go-bip39"
)

func PrivateKeyAsHex(key *ecdsa.PrivateKey) string {
	skBytes := crypto.FromECDSA(key)
	return hex.EncodeToString(skBytes)
}

func PrivateKeyAsMnemonic(key *ecdsa.PrivateKey) (string, error) {
	skBytes := crypto.FromECDSA(key)
	return bip39.NewMnemonic(skBytes)
}

func PublicKeyAsHex(key ecdsa.PublicKey) string {
	return hex.EncodeToString(secp256k1.CompressPubkey(key.X, key.Y))
}

func GenerateKey() (*ecdsa.PrivateKey, error) {
	return crypto.GenerateKey()
}

func PrivateKeyFromMnemonic(mnemonic string) (*ecdsa.PrivateKey, error) {
	e, err := bip39.EntropyFromMnemonic(mnemonic)
	if err != nil {
		return &ecdsa.PrivateKey{}, err
	}

	return crypto.ToECDSA(e)
}
