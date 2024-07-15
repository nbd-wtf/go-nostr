package nip44

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/nbd-wtf/go-nostr/nip04"
	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/hkdf"
)

const version byte = 2

const (
	MinPlaintextSize = 0x0001 // 1b msg => padded to 32b
	MaxPlaintextSize = 0xffff // 65535 (64kb-1) => padded to 64kb
)

type encryptOptions struct {
	err  error
	salt []byte
}

func WithCustomSalt(salt []byte) func(opts *encryptOptions) {
	return func(opts *encryptOptions) {
		if len(salt) != 32 {
			opts.err = errors.New("salt must be 32 bytes")
		}
		opts.salt = salt
	}
}

func Encrypt(plaintext string, conversationKey []byte, applyOptions ...func(opts *encryptOptions)) (string, error) {
	opts := encryptOptions{
		salt: nil,
	}

	for _, apply := range applyOptions {
		apply(&opts)
	}

	if opts.err != nil {
		return "", opts.err
	}

	salt := opts.salt
	if salt == nil {
		salt := make([]byte, 32)
		if _, err := rand.Read(salt); err != nil {
			return "", err
		}
	}

	enc, nonce, auth, err := messageKeys(conversationKey, salt)
	if err != nil {
		return "", err
	}

	padded, err := pad(plaintext)
	if err != nil {
		return "", err
	}

	ciphertext, err := chacha20_(enc, nonce, []byte(padded))
	if err != nil {
		return "", err
	}

	hmac_, err := sha256Hmac(auth, ciphertext, salt)
	if err != nil {
		return "", err
	}

	concat := make([]byte, 1+len(salt)+len(ciphertext)+len(hmac_))
	concat[0] = version
	copy(concat[1:], salt)
	copy(concat[1+len(salt):], ciphertext)
	copy(concat[1+len(salt)+len(ciphertext):], hmac_)

	return base64.StdEncoding.EncodeToString(concat), nil
}

func Decrypt(ciphertext string, conversationKey []byte) (string, error) {
	cLen := len(ciphertext)
	if cLen < 132 || cLen > 87472 {
		return "", errors.New(fmt.Sprintf("invalid payload length: %d", cLen))
	}
	if ciphertext[0:1] == "#" {
		return "", errors.New("unknown version")
	}

	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", errors.New("invalid base64")
	}

	if decoded[0] != version {
		return "", errors.New(fmt.Sprintf("unknown version %d", decoded[0]))
	}

	dLen := len(decoded)
	if dLen < 99 || dLen > 65603 {
		return "", errors.New(fmt.Sprintf("invalid data length: %d", dLen))
	}

	salt, ciphertext_, hmac_ := decoded[1:33], decoded[33:dLen-32], decoded[dLen-32:]
	enc, nonce, auth, err := messageKeys(conversationKey, salt)
	if err != nil {
		return "", err
	}

	hmac, err := sha256Hmac(auth, ciphertext_, salt)
	if err != nil {
		return "", err
	}

	if !bytes.Equal(hmac_, hmac) {
		return "", errors.New("invalid hmac")
	}

	padded, err := chacha20_(enc, nonce, ciphertext_)
	if err != nil {
		return "", err
	}

	unpaddedLen := binary.BigEndian.Uint16(padded[0:2])
	if unpaddedLen < uint16(MinPlaintextSize) ||
		unpaddedLen > uint16(MaxPlaintextSize) || len(padded) != 2+calcPadding(int(unpaddedLen)) {
		return "", errors.New("invalid padding")
	}

	unpadded := padded[2 : unpaddedLen+2]
	if len(unpadded) == 0 || len(unpadded) != int(unpaddedLen) {
		return "", errors.New("invalid padding")
	}

	return string(unpadded), nil
}

func GenerateConversationKey(pub string, sk string) ([]byte, error) {
	if sk >= "fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141" || sk == "0000000000000000000000000000000000000000000000000000000000000000" {
		return nil, fmt.Errorf("invalid private key: x coordinate %s is not on the secp256k1 curve", sk)
	}

	shared, err := nip04.ComputeSharedSecret(pub, sk)
	if err != nil {
		return nil, err
	}
	return hkdf.Extract(sha256.New, shared, []byte("nip44-v2")), nil
}

func chacha20_(key []byte, nonce []byte, message []byte) ([]byte, error) {
	cipher, err := chacha20.NewUnauthenticatedCipher(key, nonce)
	if err != nil {
		return nil, err
	}

	dst := make([]byte, len(message))
	cipher.XORKeyStream(dst, message)
	return dst, nil
}

func sha256Hmac(key []byte, ciphertext []byte, aad []byte) ([]byte, error) {
	if len(aad) != 32 {
		return nil, errors.New("aad data must be 32 bytes")
	}
	h := hmac.New(sha256.New, key)
	h.Write(aad)
	h.Write(ciphertext)
	return h.Sum(nil), nil
}

func messageKeys(conversationKey []byte, salt []byte) ([]byte, []byte, []byte, error) {
	if len(conversationKey) != 32 {
		return nil, nil, nil, errors.New("conversation key must be 32 bytes")
	}
	if len(salt) != 32 {
		return nil, nil, nil, errors.New("salt must be 32 bytes")
	}

	r := hkdf.Expand(sha256.New, conversationKey, salt)
	enc := make([]byte, 32)
	if _, err := io.ReadFull(r, enc); err != nil {
		return nil, nil, nil, err
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(r, nonce); err != nil {
		return nil, nil, nil, err
	}

	auth := make([]byte, 32)
	if _, err := io.ReadFull(r, auth); err != nil {
		return nil, nil, nil, err
	}

	return enc, nonce, auth, nil
}

func pad(s string) ([]byte, error) {
	sb := []byte(s)
	sbLen := len(sb)
	if sbLen < 1 || sbLen > MaxPlaintextSize {
		return nil, errors.New("plaintext should be between 1b and 64kB")
	}
	padding := calcPadding(sbLen)
	result := make([]byte, 2)
	binary.BigEndian.PutUint16(result, uint16(sbLen))
	result = append(result, sb...)
	result = append(result, make([]byte, padding-sbLen)...)
	return result, nil
}

func calcPadding(sLen int) int {
	if sLen <= 32 {
		return 32
	}
	nextPower := 1 << int(math.Floor(math.Log2(float64(sLen-1)))+1)
	chunk := int(math.Max(32, float64(nextPower/8)))
	return chunk * int(math.Floor(float64((sLen-1)/chunk))+1)
}
