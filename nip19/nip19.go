package nip19

import (
	"encoding/binary"
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

func EncodePublicKey(publicKeyHex string, masterRelay string) (string, error) {
	b, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return "", err
	}

	tlv := make([]byte, 0, 64)
	if masterRelay != "" {
		relayBytes := []byte(masterRelay)
		length := len(relayBytes)
		if length >= 65536 {
			return "", fmt.Errorf("masterRelay URL is too large")
		}

		binary.BigEndian.PutUint16(tlv, 1)
		binary.BigEndian.PutUint16(tlv, uint16(length))
		tlv = append(tlv, relayBytes...)
	}
	b = append(b, tlv...)

	return encode("nsec", b)
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

	if len(data) < 32 {
		return nil, "", fmt.Errorf("data is less than 32 bytes (%d)", len(data))
	}

	return data[0:32], prefix, nil
}
