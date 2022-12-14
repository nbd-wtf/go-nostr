package nip04

import (
	"strings"
	"testing"
)

func TestEncryptionAndDecryption(t *testing.T) {
	sharedSecret := make([]byte, 32)
	message := "hello hellow"

	ciphertext, err := Encrypt(message, sharedSecret)
	if err != nil {
		t.Errorf("failed to encrypt: %s", err.Error())
	}

	plaintext, err := Decrypt(ciphertext, sharedSecret)
	if err != nil {
		t.Errorf("failed to decrypt: %s", err.Error())
	}

	if message != plaintext {
		t.Errorf("original '%s' and decrypted '%s' messages differ", message, plaintext)
	}
}

func TestEncryptionAndDecryptionWithMultipleLengths(t *testing.T) {
	sharedSecret := make([]byte, 32)

	for i := 0; i < 150; i++ {
		message := strings.Repeat("a", i)

		ciphertext, err := Encrypt(message, sharedSecret)
		if err != nil {
			t.Errorf("failed to encrypt: %s", err.Error())
		}

		plaintext, err := Decrypt(ciphertext, sharedSecret)
		if err != nil {
			t.Errorf("failed to decrypt: %s", err.Error())
		}

		if message != plaintext {
			t.Errorf("original '%s' and decrypted '%s' messages differ", message, plaintext)
		}
	}
}
