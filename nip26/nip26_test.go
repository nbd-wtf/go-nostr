package nip26

import (
	"testing"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

func TestDelegateSign(t *testing.T) {
	since := time.Unix(1600000000, 0)
	until := time.Unix(1600000100, 0)
	delegator_secret_key, delegatee_secret_key := "3f0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459da", "e9142f724955c5854de36324dab0434f97b15ec6b33464d56ebe491e3f559d1b"
	delegatee_pubkey, _ := nostr.GetPublicKey(delegatee_secret_key)
	d1, err := CreateToken(delegator_secret_key, delegatee_pubkey, []int{1, 2, 3}, &since, &until)
	if err != nil {
		t.Error(err)
	}
	ev := &nostr.Event{}
	ev.CreatedAt = nostr.Timestamp(1600000050)
	ev.Content = "hello world"
	ev.Kind = 1
	if err != nil {
		t.Error(err)
	}
	d2, err := Import(d1.Tag(), delegatee_pubkey)
	if err != nil {
		t.Error(err)
	}

	if err = DelegatedSign(ev, d2, delegatee_secret_key); err != nil {
		t.Error(err)
	}
	if ok, err := CheckDelegation(ev); err != nil || ok == false {
		t.Error(err)
	}

	tag := []string{"delegation", "9ea72be3fcfe38103195a41b67b6f96c14ed92d2091d6d9eb8166a5c27b0c35d", "kind=1&kind=2&kind=3&created_at>1600000000", "8432b8c86f789c2783ef3becb0fabf4def6031c6a615fa7a622f31329d80ed1b2a79ab753c0462f1440503c94e1829158a3a854a1d418ad256ae2cf8aa19fa9a"}
	d3, err := Import(tag, delegatee_pubkey)
	if err != nil {
		t.Error(err)
	}

	ev.Tags = nil
	if err = DelegatedSign(ev, d3, delegatee_secret_key); err != nil {
		t.Error(err)
	}
	if ok, err := d3.Parse(ev); err != nil || ok == false {
		t.Error(err)
	}
}
