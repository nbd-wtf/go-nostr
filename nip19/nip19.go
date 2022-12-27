package nip19

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

type ProfilePointer struct {
	PublicKey string
	Relays    []string
}

type EventPointer struct {
	ID     string
	Relays []string
}

func Decode(bech32string string) (prefix string, value any, err error) {
	prefix, bits5, err := decode(bech32string)
	if err != nil {
		return "", nil, err
	}

	data, err := convertBits(bits5, 5, 8, false)
	if err != nil {
		return prefix, nil, fmt.Errorf("failed translating data into 8 bits: %s", err.Error())
	}

	switch prefix {
	case "npub", "nsec", "note":
		if len(data) < 32 {
			return prefix, nil, fmt.Errorf("data is less than 32 bytes (%d)", len(data))
		}

		return prefix, hex.EncodeToString(data[0:32]), nil
	case "nprofile":
		var result ProfilePointer
		curr := 0
		for {
			t, v := readTLVEntry(data[curr:])
			if v == nil {
				// end here
				if result.PublicKey == "" {
					return prefix, result, fmt.Errorf("no pubkey found for nprofile")
				}

				return prefix, result, nil
			}

			switch t {
			case TLVDefault:
				result.PublicKey = hex.EncodeToString(v)
			case TLVRelay:
				result.Relays = append(result.Relays, string(v))
			default:
				// ignore
			}

			curr = curr + 2 + len(v)
		}
	case "nevent":
		var result EventPointer
		curr := 0
		for {
			t, v := readTLVEntry(data[curr:])
			if v == nil {
				// end here
				if result.ID == "" {
					return prefix, result, fmt.Errorf("no id found for nevent")
				}

				return prefix, result, nil
			}

			switch t {
			case TLVDefault:
				result.ID = hex.EncodeToString(v)
			case TLVRelay:
				result.Relays = append(result.Relays, string(v))
			default:
				// ignore
			}

			curr = curr + 2 + len(v)
		}
	}

	return prefix, data, fmt.Errorf("unknown tag %s", prefix)
}

func EncodePrivateKey(privateKeyHex string) (string, error) {
	b, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key hex: %w", err)
	}

	bits5, err := convertBits(b, 8, 5, true)
	if err != nil {
		return "", err
	}

	return encode("nsec", bits5)
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
		return "", fmt.Errorf("failed to decode event id hex: %w", err)
	}

	bits5, err := convertBits(b, 8, 5, true)
	if err != nil {
		return "", err
	}

	return encode("note", bits5)
}

func EncodeProfile(publicKeyHex string, relays []string) (string, error) {
	buf := &bytes.Buffer{}
	pubkey, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid pubkey '%s': %w", publicKeyHex, err)
	}
	writeTLVEntry(buf, TLVDefault, pubkey)

	for _, url := range relays {
		writeTLVEntry(buf, TLVRelay, []byte(url))
	}

	bits5, err := convertBits(buf.Bytes(), 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("failed to convert bits: %w", err)
	}

	return encode("nprofile", bits5)
}

func EncodeEvent(eventIdHex string, relays []string) (string, error) {
	buf := &bytes.Buffer{}
	pubkey, err := hex.DecodeString(eventIdHex)
	if err != nil {
		return "", fmt.Errorf("invalid id '%s': %w", eventIdHex, err)
	}
	writeTLVEntry(buf, TLVDefault, pubkey)

	for _, url := range relays {
		writeTLVEntry(buf, TLVRelay, []byte(url))
	}

	bits5, err := convertBits(buf.Bytes(), 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("failed to convert bits: %w", err)
	}

	return encode("nevent", bits5)
}
