package nip19

import (
	"fmt"
	"testing"
)

func TestEncodeNpub(t *testing.T) {
	npub, err := EncodePublicKey("3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d")
	if err != nil {
		t.Errorf("shouldn't error: %s", err)
	}
	if npub != "npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6" {
		t.Error("produced an unexpected npub string")
	}
}

func TestEncodeNsec(t *testing.T) {
	npub, err := EncodePrivateKey("3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d")
	if err != nil {
		t.Errorf("shouldn't error: %s", err)
	}
	if npub != "nsec180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsgyumg0" {
		t.Error("produced an unexpected nsec string")
	}
}

func TestDecodeNpub(t *testing.T) {
	prefix, pubkey, err := Decode("npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6")
	if err != nil {
		t.Errorf("shouldn't error: %s", err)
	}
	if prefix != "npub" {
		t.Error("returned invalid prefix")
	}
	if pubkey.(string) != "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d" {
		t.Error("returned wrong pubkey")
	}
}

func TestFailDecodeBadChecksumNpub(t *testing.T) {
	_, _, err := Decode("npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w4")
	if err == nil {
		t.Errorf("should have errored: %s", err)
	}
}

func TestDecodeNprofile(t *testing.T) {
	prefix, data, err := Decode("nprofile1qqsrhuxx8l9ex335q7he0f09aej04zpazpl0ne2cgukyawd24mayt8gpp4mhxue69uhhytnc9e3k7mgpz4mhxue69uhkg6nzv9ejuumpv34kytnrdaksjlyr9p")
	if err != nil {
		t.Error("failed to decode nprofile")
	}
	if prefix != "nprofile" {
		t.Error("what")
	}
	pp, ok := data.(ProfilePointer)
	if !ok {
		t.Error("value returned of wrong type")
	}

	fmt.Println(pp)

	if pp.PublicKey != "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d" {
		t.Error("decoded invalid public key")
	}

	if len(pp.Relays) != 2 {
		t.Error("decoded wrong number of relays")
	}
	if pp.Relays[0] != "wss://r.x.com" || pp.Relays[1] != "wss://djbas.sadkb.com" {
		t.Error("decoded relay URLs wrongly")
	}
}

func TestEncodeNprofile(t *testing.T) {
	nprofile, err := EncodeProfile("3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d", []string{
		"wss://r.x.com",
		"wss://djbas.sadkb.com",
	})
	if err != nil {
		t.Errorf("shouldn't error: %s", err)
	}
	if nprofile != "nprofile1qqsrhuxx8l9ex335q7he0f09aej04zpazpl0ne2cgukyawd24mayt8gpp4mhxue69uhhytnc9e3k7mgpz4mhxue69uhkg6nzv9ejuumpv34kytnrdaksjlyr9p" {
		t.Error("produced an unexpected nprofile string")
	}
}
