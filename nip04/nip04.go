package nip04

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcec"
)

// ECDH
func ComputeSharedSecret(senderPrivKey string, receiverPubKey string) (sharedSecret []byte, err error) {
	privKeyBytes, err := hex.DecodeString(senderPrivKey)
	if err != nil {
		return nil, fmt.Errorf("Error decoding sender private key: %s. \n", err)
	}
	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKeyBytes)

	// adding 02 to signal that this is a compressed public key (33 bytes)
	pubKeyBytes, err := hex.DecodeString("02" + receiverPubKey)
	if err != nil {
		return nil, fmt.Errorf("Error decoding hex string of receiver public key: %s. \n", err)
	}
	pubKey, err := btcec.ParsePubKey(pubKeyBytes, btcec.S256())
	if err != nil {
		return nil, fmt.Errorf("Error parsing receiver public key: %s. \n", err)
	}

	return btcec.GenerateSharedSecret(privKey, pubKey), nil
}

// aes-256-cbc
func Encrypt(message string, key []byte) (string, error) {
	// block size is 16 bytes
	iv := make([]byte, 16)
	// can probably use a less expensive lib since IV has to only be unique; not perfectly random; math/rand?
	_, err := rand.Read(iv)
	if err != nil {
		return "", fmt.Errorf("Error creating initization vector: %s. \n", err.Error())
	}

	// automatically picks aes-256 based on key length (32 bytes)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("Error creating block cipher: %s. \n", err.Error())
	}
	mode := cipher.NewCBCEncrypter(block, iv)

	// PKCS5 padding
	padding := block.BlockSize() - len([]byte(message))%block.BlockSize()
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	paddedMsgBytes := append([]byte(message), padtext...)

	ciphertext := make([]byte, len(paddedMsgBytes))
	mode.CryptBlocks(ciphertext, paddedMsgBytes)

	return base64.StdEncoding.EncodeToString(ciphertext) + "?iv=" + base64.StdEncoding.EncodeToString(iv), nil
}

// aes-256-cbc
func Decrypt(content string, key []byte) (string, error) {
	parts := strings.Split(content, "?iv=")
	if len(parts) < 2 {
		return "", fmt.Errorf("Error parsing encrypted message: no initilization vector. \n")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("Error decoding ciphertext from base64: %s. \n", err)
	}

	iv, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("Error decoding iv from base64: %s. \n", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("Error creating block cipher: %s. \n", err.Error())
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	return string(plaintext[:]), nil
}
