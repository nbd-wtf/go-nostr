package nip49

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestDecryptKeyFromNIPText(t *testing.T) {
	ncrypt := "ncryptsec1qgg9947rlpvqu76pj5ecreduf9jxhselq2nae2kghhvd5g7dgjtcxfqtd67p9m0w57lspw8gsq6yphnm8623nsl8xn9j4jdzz84zm3frztj3z7s35vpzmqf6ksu8r89qk5z2zxfmu5gv8th8wclt0h4p"
	secretKey, err := Decrypt(ncrypt, "nostr")
	if err != nil {
		t.Fatalf("failed to decrypt: %s", err)
	}
	if secretKey != "3501454135014541350145413501453fefb02227e449e57cf4d3a3ce05378683" {
		t.Fatalf("decrypted wrongly: %s", secretKey)
	}
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
		{"", "11b25a101667dd9208db93c0827c6bdad66729a5b521156a7e9d3b22b3ae8944", 9, 0x01},
	} {
		bech32code, err := Encrypt(f.secretkey, f.password, f.logn, f.ksb)
		if err != nil {
			t.Fatalf("failed to encrypt %d: %s", i, err)
		}
		if !strings.HasPrefix(bech32code, "ncryptsec1") || len(bech32code) != 162 {
			t.Fatalf("bech32 code is wrong %d: %s", i, bech32code)
		}

		secretKey, err := Decrypt(bech32code, f.password)
		if err != nil {
			t.Fatalf("failed to decrypt %d: %s", i, err)
		}
		if secretKey != f.secretkey {
			t.Fatalf("decrypted to the wrong value %d: %s", i, secretKey)
		}
	}
}

func TestNormalization(t *testing.T) {
	nonce := []byte{1, 2, 3, 4}
	n := 8
	key1, err1 := getKey(string([]byte{0xE2, 0x84, 0xAB, 0xE2, 0x84, 0xA6}), nonce, n)
	key2, err2 := getKey(string([]byte{0xC3, 0x85, 0xCE, 0xA9}), nonce, n)
	key3, err3 := getKey("ÅΩ", nonce, n)
	key4, err4 := getKey("ÅΩ", nonce, n)
	if merr := errors.Join(err1, err2, err3, err4); merr != nil {
		t.Fatalf("getKey errored: %s", merr)
		return
	}

	if !slices.Equal(key1, key2) || !slices.Equal(key2, key3) || !slices.Equal(key3, key4) {
		t.Fatalf("normalization failed")
		return
	}
}
