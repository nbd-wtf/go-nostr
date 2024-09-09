package nip49

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecryptKeyFromNIPText(t *testing.T) {
	ncrypt := "ncryptsec1qgg9947rlpvqu76pj5ecreduf9jxhselq2nae2kghhvd5g7dgjtcxfqtd67p9m0w57lspw8gsq6yphnm8623nsl8xn9j4jdzz84zm3frztj3z7s35vpzmqf6ksu8r89qk5z2zxfmu5gv8th8wclt0h4p"
	secretKey, err := Decrypt(ncrypt, "nostr")
	assert.NoError(t, err)
	assert.Equal(t, "3501454135014541350145413501453fefb02227e449e57cf4d3a3ce05378683", secretKey)
}

func TestEncryptAndDecrypt(t *testing.T) {
	for i, f := range []struct {
		password  string
		secretkey string
		logn      uint8
		ksb       KeySecurityByte
	}{
		{".ksjabdk.aselqwe", "14c226dbdd865d5e1645e72c7470fd0a17feb42cc87b750bab6538171b3a3f8a", 1, 0x00},
		{"skjdaklrnçurbç l", "f7f2f77f98890885462764afb15b68eb5f69979c8046ecb08cad7c4ae6b221ab", 2, 0x01},
		{"777z7z7z7z7z7z7z", "11b25a101667dd9208db93c0827c6bdad66729a5b521156a7e9d3b22b3ae8944", 3, 0x02},
		{".ksjabdk.aselqwe", "14c226dbdd865d5e1645e72c7470fd0a17feb42cc87b750bab6538171b3a3f8a", 7, 0x00},
		{"skjdaklrnçurbç l", "f7f2f77f98890885462764afb15b68eb5f69979c8046ecb08cad7c4ae6b221ab", 8, 0x01},
		{"777z7z7z7z7z7z7z", "11b25a101667dd9208db93c0827c6bdad66729a5b521156a7e9d3b22b3ae8944", 9, 0x02},
		{"", "f7f2f77f98890885462764afb15b68eb5f69979c8046ecb08cad7c4ae6b221ab", 4, 0x00},
		{"", "11b25a101667dd9208db93c0827c6bdad66729a5b521156a7e9d3b22b3ae8944", 5, 0x01},
		{"", "f7f2f77f98890885462764afb15b68eb5f69979c8046ecb08cad7c4ae6b221ab", 1, 0x00},
		{"ÅΩẛ̣", "11b25a101667dd9208db93c0827c6bdad66729a5b521156a7e9d3b22b3ae8944", 9, 0x01},
		{"ÅΩṩ", "11b25a101667dd9208db93c0827c6bdad66729a5b521156a7e9d3b22b3ae8944", 9, 0x01},
	} {
		bech32code, err := Encrypt(f.secretkey, f.password, f.logn, f.ksb)
		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(bech32code, "ncryptsec1"), "bech32 code is wrong %d: %s", i, bech32code)
		assert.Equal(t, 162, len(bech32code), "bech32 code is wrong %d: %s", i, bech32code)

		secretKey, err := Decrypt(bech32code, f.password)
		assert.NoError(t, err)
		assert.Equal(t, f.secretkey, secretKey)
	}
}

func TestNormalization(t *testing.T) {
	nonce := []byte{1, 2, 3, 4}
	n := 8
	key1, err1 := getKey(string([]byte{0xE2, 0x84, 0xAB, 0xE2, 0x84, 0xA6, 0xE1, 0xBA, 0x9B, 0xCC, 0xA3}), nonce, n)
	key2, err2 := getKey(string([]byte{0xC3, 0x85, 0xCE, 0xA9, 0xE1, 0xB9, 0xA9}), nonce, n)
	key3, err3 := getKey("ÅΩẛ̣", nonce, n)
	key4, err4 := getKey("ÅΩẛ̣", nonce, n)
	err := errors.Join(err1, err2, err3, err4)
	assert.NoError(t, err)
	assert.True(t, slices.Equal(key1, key2), "normalization failed")
	assert.True(t, slices.Equal(key2, key3), "normalization failed")
	assert.True(t, slices.Equal(key3, key4), "normalization failed")
}
