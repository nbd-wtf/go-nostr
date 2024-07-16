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
	err   error
	nonce []byte
}

// Deprecated: use WithCustomNonce instead of WithCustomSalt, so the naming is less confusing
var WithCustomSalt = WithCustomNonce

func WithCustomNonce(salt []byte) func(opts *encryptOptions) {
	return func(opts *encryptOptions) {
		if len(salt) != 32 {
			opts.err = errors.New("salt must be 32 bytes")
		}
		opts.nonce = salt
	}
}

func Encrypt(plaintext string, conversationKey []byte, applyOptions ...func(opts *encryptOptions)) (string, error) {
	opts := encryptOptions{}
	for _, apply := range applyOptions {
		apply(&opts)
	}

	if opts.err != nil {
		return "", opts.err
	}

	nonce := opts.nonce
	if nonce == nil {
		nonce := make([]byte, 32)
		if _, err := rand.Read(nonce); err != nil {
			return "", err
		}
	}

	enc, cc20nonce, auth, err := messageKeys(conversationKey, nonce)
	if err != nil {
		return "", err
	}

	plain := []byte(plaintext)
	size := len(plain)
	if size < MinPlaintextSize || size > MaxPlaintextSize {
		return "", errors.New("plaintext should be between 1b and 64kB")
	}

	padding := calcPadding(size)
	padded := make([]byte, 2+padding)
	binary.BigEndian.PutUint16(padded, uint16(size))
	copy(padded[2:], plain)

	ciphertext, err := chacha(enc, cc20nonce, []byte(padded))
	if err != nil {
		return "", err
	}

	mac, err := sha256Hmac(auth, ciphertext, nonce)
	if err != nil {
		return "", err
	}

	concat := make([]byte, 1+32+len(ciphertext)+32)
	concat[0] = version
	copy(concat[1:], nonce)
	copy(concat[1+32:], ciphertext)
	copy(concat[1+32+len(ciphertext):], mac)

	return base64.StdEncoding.EncodeToString(concat), nil
}

func Decrypt(b64ciphertextWrapped string, conversationKey []byte) (string, error) {
	cLen := len(b64ciphertextWrapped)
	if cLen < 132 || cLen > 87472 {
		return "", errors.New(fmt.Sprintf("invalid payload length: %d", cLen))
	}
	if b64ciphertextWrapped[0:1] == "#" {
		return "", errors.New("unknown version")
	}

	decoded, err := base64.StdEncoding.DecodeString(b64ciphertextWrapped)
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

	nonce, ciphertext, givenMac := decoded[1:33], decoded[33:dLen-32], decoded[dLen-32:]
	enc, cc20nonce, auth, err := messageKeys(conversationKey, nonce)
	if err != nil {
		return "", err
	}

	expectedMac, err := sha256Hmac(auth, ciphertext, nonce)
	if err != nil {
		return "", err
	}

	if !bytes.Equal(givenMac, expectedMac) {
		return "", errors.New("invalid hmac")
	}

	padded, err := chacha(enc, cc20nonce, ciphertext)
	if err != nil {
		return "", err
	}

	unpaddedLen := binary.BigEndian.Uint16(padded[0:2])
	if unpaddedLen < uint16(MinPlaintextSize) || unpaddedLen > uint16(MaxPlaintextSize) ||
		len(padded) != 2+calcPadding(int(unpaddedLen)) {
		return "", errors.New("invalid padding")
	}

	unpadded := padded[2:][:unpaddedLen]
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

func chacha(key []byte, nonce []byte, message []byte) ([]byte, error) {
	cipher, err := chacha20.NewUnauthenticatedCipher(key, nonce)
	if err != nil {
		return nil, err
	}

	dst := make([]byte, len(message))
	cipher.XORKeyStream(dst, message)
	return dst, nil
}

func sha256Hmac(key []byte, ciphertext []byte, nonce []byte) ([]byte, error) {
	if len(nonce) != 32 {
		return nil, errors.New("nonce aad must be 32 bytes")
	}
	h := hmac.New(sha256.New, key)
	h.Write(nonce)
	h.Write(ciphertext)
	return h.Sum(nil), nil
}

func messageKeys(conversationKey []byte, nonce []byte) ([]byte, []byte, []byte, error) {
	if len(conversationKey) != 32 {
		return nil, nil, nil, errors.New("conversation key must be 32 bytes")
	}
	if len(nonce) != 32 {
		return nil, nil, nil, errors.New("nonce must be 32 bytes")
	}

	r := hkdf.Expand(sha256.New, conversationKey, nonce)
	enc := make([]byte, 32)
	if _, err := io.ReadFull(r, enc); err != nil {
		return nil, nil, nil, err
	}

	cc20nonce := make([]byte, 12)
	if _, err := io.ReadFull(r, cc20nonce); err != nil {
		return nil, nil, nil, err
	}

	auth := make([]byte, 32)
	if _, err := io.ReadFull(r, auth); err != nil {
		return nil, nil, nil, err
	}

	return enc, cc20nonce, auth, nil
}

func calcPadding(sLen int) int {
	if sLen <= 32 {
		return 32
	}
	nextPower := 1 << int(math.Floor(math.Log2(float64(sLen-1)))+1)
	chunk := int(math.Max(32, float64(nextPower/8)))
	return chunk * int(math.Floor(float64((sLen-1)/chunk))+1)
}
