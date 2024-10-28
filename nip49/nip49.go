package nip49

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"

	"github.com/btcsuite/btcd/btcutil/bech32"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/scrypt"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type KeySecurityByte byte

const (
	KnownToHaveBeenHandledInsecurely    KeySecurityByte = 0x00
	NotKnownToHaveBeenHandledInsecurely KeySecurityByte = 0x01
	ClientDoesNotTrackThisData          KeySecurityByte = 0x02
)

func Encrypt(secretKey string, password string, logn uint8, ksb KeySecurityByte) (b32code string, err error) {
	skb, err := hex.DecodeString(secretKey)
	if err != nil || len(skb) != 32 {
		return "", fmt.Errorf("invalid secret key")
	}
	return EncryptBytes(skb, password, logn, ksb)
}

func EncryptBytes(secretKey []byte, password string, logn uint8, ksb KeySecurityByte) (b32code string, err error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to read salt: %w", err)
	}
	n := int(math.Pow(2, float64(int(logn))))

	key, err := getKey(password, salt, n)
	if err != nil {
		return "", err
	}

	concat := make([]byte, 91)
	concat[0] = 0x02
	concat[1] = byte(logn)
	copy(concat[2:2+16], salt)
	rand.Read(concat[2+16 : 2+16+24]) // nonce
	ad := []byte{byte(ksb)}
	copy(concat[2+16+24:2+16+24+1], ad)

	c2p1, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", fmt.Errorf("failed to start xchacha20poly1305: %w", err)
	}
	ciphertext := c2p1.Seal(nil, concat[2+16:2+16+24], secretKey, ad)
	copy(concat[2+16+24+1:], ciphertext)

	bits5, err := bech32.ConvertBits(concat, 8, 5, true)
	if err != nil {
		return "", err
	}
	return bech32.Encode("ncryptsec", bits5)
}

func Decrypt(bech32string string, password string) (secretKey string, err error) {
	secb, err := DecryptToBytes(bech32string, password)
	return hex.EncodeToString(secb), err
}

func DecryptToBytes(bech32string string, password string) (secretKey []byte, err error) {
	prefix, bits5, err := bech32.DecodeNoLimit(bech32string)
	if err != nil {
		return nil, err
	}
	if prefix != "ncryptsec" {
		return nil, fmt.Errorf("expected prefix ncryptsec1")
	}

	data, err := bech32.ConvertBits(bits5, 5, 8, false)
	if err != nil {
		return nil, fmt.Errorf("failed translating data into 8 bits: %s", err.Error())
	}

	version := data[0]
	if version != 0x02 {
		return nil, fmt.Errorf("expected version 0x02, got %v", version)
	}

	logn := data[1]
	n := int(math.Pow(2, float64(int(logn))))
	salt := data[2 : 2+16]
	nonce := data[2+16 : 2+16+24]
	ad := data[2+16+24 : 2+16+24+1]
	// keySecurityByte := ad[0]
	encryptedKey := data[2+16+24+1:]

	key, err := getKey(password, salt, n)
	if err != nil {
		return nil, err
	}

	c2p1, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("failed to start xchacha20poly1305: %w", err)
	}

	return c2p1.Open(nil, nonce, encryptedKey, ad)
}

func getKey(password string, salt []byte, n int) ([]byte, error) {
	normalizedPassword, _, err := transform.Bytes(norm.NFKC, []byte(password))
	if err != nil {
		return nil, fmt.Errorf("failed to normalize password: %w", err)
	}

	key, err := scrypt.Key(normalizedPassword, salt, n, 8, 1, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to compute key with scrypt: %w", err)
	}
	return key, nil
}
