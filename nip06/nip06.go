package nip06

import (
	"encoding/hex"

	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

func GenerateSeedWords() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}

	words, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", err
	}

	return words, nil
}

func SeedFromWords(words string) []byte {
	return bip39.NewSeed(words, "")
}

func PrivateKeyFromSeed(seed []byte) (string, error) {
	key, err := bip32.NewMasterKey(seed)
	if err != nil {
		return "", err
	}

	derivationPath := []uint32{
		bip32.FirstHardenedChild + 44,
		bip32.FirstHardenedChild + 1237,
		bip32.FirstHardenedChild + 0,
		0,
		0,
	}

	next := key
	for _, idx := range derivationPath {
		var err error
		if next, err = next.NewChildKey(idx); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(next.Key), nil
}

func ValidateWords(words string) bool {
	return bip39.IsMnemonicValid(words)
}
