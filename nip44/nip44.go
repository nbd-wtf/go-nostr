package nip44

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/hkdf"
)

const version byte = 2

const (
	MinPlaintextSize = 0x0001 // 1b msg => padded to 32b
	MaxPlaintextSize = 0xffff // 65535 (64kb-1) => padded to 64kb
)

var zeroes = [32]byte{}

type encryptOptions struct {
	err   error
	nonce [32]byte
}

func WithCustomNonce(nonce []byte) func(opts *encryptOptions) {
	return func(opts *encryptOptions) {
		if len(nonce) != 32 {
			opts.err = fmt.Errorf("invalid custom nonce, must be 32 bytes, got %d", len(nonce))
		}
		copy(opts.nonce[:], nonce)
	}
}

func Encrypt(plaintext string, conversationKey [32]byte, applyOptions ...func(opts *encryptOptions)) (string, error) {
	opts := encryptOptions{}
	for _, apply := range applyOptions {
		apply(&opts)
	}

	if opts.err != nil {
		return "", opts.err
	}

	nonce := opts.nonce
	if nonce == zeroes {
		if _, err := rand.Read(nonce[:]); err != nil {
			return "", err
		}
	}

	cc20key, cc20nonce, hmackey, err := messageKeys(conversationKey, nonce)
	if err != nil {
		return "", err
	}

	plain := []byte(plaintext)
	size := len(plain)
	if size < MinPlaintextSize || size > MaxPlaintextSize {
		return "", fmt.Errorf("plaintext should be between 1b and 64kB")
	}

	padding := calcPadding(size)
	padded := make([]byte, 2+padding)
	binary.BigEndian.PutUint16(padded, uint16(size))
	copy(padded[2:], plain)

	ciphertext, err := chacha(cc20key, cc20nonce, []byte(padded))
	if err != nil {
		return "", err
	}

	mac, err := sha256Hmac(hmackey, ciphertext, nonce)
	if err != nil {
		return "", err
	}

	concat := make([]byte, 1+32+len(ciphertext)+32)
	concat[0] = version
	copy(concat[1:], nonce[:])
	copy(concat[1+32:], ciphertext)
	copy(concat[1+32+len(ciphertext):], mac)

	return base64.StdEncoding.EncodeToString(concat), nil
}

func Decrypt(b64ciphertextWrapped string, conversationKey [32]byte) (string, error) {
	cLen := len(b64ciphertextWrapped)
	if cLen < 132 || cLen > 87472 {
		return "", fmt.Errorf("invalid payload length: %d", cLen)
	}
	if b64ciphertextWrapped[0:1] == "#" {
		return "", fmt.Errorf("unknown version")
	}

	decoded, err := base64.StdEncoding.DecodeString(b64ciphertextWrapped)
	if err != nil {
		return "", fmt.Errorf("invalid base64: %w", err)
	}

	if decoded[0] != version {
		return "", fmt.Errorf("unknown version %d", decoded[0])
	}

	dLen := len(decoded)
	if dLen < 99 || dLen > 65603 {
		return "", fmt.Errorf("invalid data length: %d", dLen)
	}

	var nonce [32]byte
	copy(nonce[:], decoded[1:33])
	ciphertext := decoded[33 : dLen-32]
	givenMac := decoded[dLen-32:]
	cc20key, cc20nonce, hmackey, err := messageKeys(conversationKey, nonce)
	if err != nil {
		return "", err
	}

	expectedMac, err := sha256Hmac(hmackey, ciphertext, nonce)
	if err != nil {
		return "", err
	}

	if !bytes.Equal(givenMac, expectedMac) {
		return "", fmt.Errorf("invalid hmac")
	}

	padded, err := chacha(cc20key, cc20nonce, ciphertext)
	if err != nil {
		return "", err
	}

	unpaddedLen := binary.BigEndian.Uint16(padded[0:2])
	if unpaddedLen < uint16(MinPlaintextSize) || unpaddedLen > uint16(MaxPlaintextSize) ||
		len(padded) != 2+calcPadding(int(unpaddedLen)) {
		return "", fmt.Errorf("invalid padding")
	}

	unpadded := padded[2:][:unpaddedLen]
	if len(unpadded) == 0 || len(unpadded) != int(unpaddedLen) {
		return "", fmt.Errorf("invalid padding")
	}

	return string(unpadded), nil
}

func GenerateConversationKey(pub string, sk string) ([32]byte, error) {
	var ck [32]byte

	if sk >= "fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141" || sk == "0000000000000000000000000000000000000000000000000000000000000000" {
		return ck, fmt.Errorf("invalid private key: x coordinate %s is not on the secp256k1 curve", sk)
	}

	shared, err := computeSharedSecret(pub, sk)
	if err != nil {
		return ck, err
	}

	buf := hkdf.Extract(sha256.New, shared[:], []byte("nip44-v2"))
	copy(ck[:], buf)

	return ck, nil
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

func sha256Hmac(key []byte, ciphertext []byte, nonce [32]byte) ([]byte, error) {
	h := hmac.New(sha256.New, key)
	h.Write(nonce[:])
	h.Write(ciphertext)
	return h.Sum(nil), nil
}

func messageKeys(conversationKey [32]byte, nonce [32]byte) ([]byte, []byte, []byte, error) {
	r := hkdf.Expand(sha256.New, conversationKey[:], nonce[:])

	cc20key := make([]byte, 32)
	if _, err := io.ReadFull(r, cc20key); err != nil {
		return nil, nil, nil, err
	}

	cc20nonce := make([]byte, 12)
	if _, err := io.ReadFull(r, cc20nonce); err != nil {
		return nil, nil, nil, err
	}

	hmacKey := make([]byte, 32)
	if _, err := io.ReadFull(r, hmacKey); err != nil {
		return nil, nil, nil, err
	}

	return cc20key, cc20nonce, hmacKey, nil
}

func calcPadding(sLen int) int {
	if sLen <= 32 {
		return 32
	}
	nextPower := 1 << int(math.Floor(math.Log2(float64(sLen-1)))+1)
	chunk := int(math.Max(32, float64(nextPower/8)))
	return chunk * int(math.Floor(float64((sLen-1)/chunk))+1)
}

// code adapted from nip04.ComputeSharedSecret()
func computeSharedSecret(pub string, sk string) (sharedSecret [32]byte, err error) {
	privKeyBytes, err := hex.DecodeString(sk)
	if err != nil {
		return sharedSecret, fmt.Errorf("error decoding sender private key: %w", err)
	}
	privKey, _ := btcec.PrivKeyFromBytes(privKeyBytes)

	pubKeyBytes, err := hex.DecodeString("02" + pub)
	if err != nil {
		return sharedSecret, fmt.Errorf("error decoding hex string of receiver public key '%s': %w", "02"+pub, err)
	}
	pubKey, err := btcec.ParsePubKey(pubKeyBytes)
	if err != nil {
		return sharedSecret, fmt.Errorf("error parsing receiver public key '%s': %w", "02"+pub, err)
	}

	var point, result secp256k1.JacobianPoint
	pubKey.AsJacobian(&point)
	secp256k1.ScalarMultNonConst(&privKey.Key, &point, &result)
	result.ToAffine()

	result.X.PutBytesUnchecked(sharedSecret[:])
	return sharedSecret, nil
}
