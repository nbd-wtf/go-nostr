package nip26

import (
	"github.com/nbd-wtf/go-nostr"
	"testing"
	"time"
)

func TestDelegateSign(t *testing.T) {
	since := time.Unix(1600000000, 0)
	delegator_secret_key, delegatee_secret_key := "3f0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459da", "e9142f724955c5854de36324dab0434f97b15ec6b33464d56ebe491e3f559d1b"
	delegatee_pubkey, _ := nostr.GetPublicKey(delegatee_secret_key)
	d1, err := CreateToken(delegator_secret_key, delegatee_pubkey, []int{1, 2, 3}, &since, nil)
	if err != nil {
		t.Error(err)
	}
	ev := &nostr.Event{}
	ev.CreatedAt = time.Now()
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

	tag := []string{"delegation", "9ea72be3fcfe38103195a41b67b6f96c14ed92d2091d6d9eb8166a5c27b0c35d", "created_at>1600000000", "c9c71d249455237c0fb620f5d1d271c1b937c1c10ee96ba1932737b0ffc3cfd49ebd918393bf4154ee867f919e56854d7b55adb19357e484d47c1f07ad29e9a9"}
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
