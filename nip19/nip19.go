package nip19

import (
	"encoding/hex"
	"fmt"
)

func EncodePrivateKey(privateKeyHex string) (string, error) {
	b, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", err
	}

	return encode("nsec", b)
}

func EncodePublicKey(publicKeyHex string) (string, error) {
	b, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode public key hex: %w", err)
	}

	bits5, err := convertBits(b, 8, 5, true)
	if err != nil {
		return "", err
	}

	return encode("npub", bits5)
}

func EncodeNote(eventIdHex string) (string, error) {
	b, err := hex.DecodeString(eventIdHex)
	if err != nil {
		return "", err
	}

	return encode("note", b)
}

func Decode(bech32string string) ([]byte, string, error) {
	prefix, data, err := decode(bech32string)
	if err != nil {
		return nil, "", err
	}

	bits8, err := convertBits(data, 5, 8, false)
	if err != nil {
		return nil, "", fmt.Errorf("failed translating data into 8 bits: %s", err.Error())
	}

	if len(data) < 32 {
		return nil, "", fmt.Errorf("data is less than 32 bytes (%d)", len(data))
	}

	return bits8[0:32], prefix, nil
}
