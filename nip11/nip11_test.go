package nip11

import (
	"context"
	"testing"
)

func TestAddSupportedNIP(t *testing.T) {
	info := RelayInformationDocument{}
	info.AddSupportedNIP(12)
	info.AddSupportedNIP(12)
	info.AddSupportedNIP(13)
	info.AddSupportedNIP(1)
	info.AddSupportedNIP(12)
	info.AddSupportedNIP(44)
	info.AddSupportedNIP(2)
	info.AddSupportedNIP(13)
	info.AddSupportedNIP(2)
	info.AddSupportedNIP(13)
	info.AddSupportedNIP(0)
	info.AddSupportedNIP(17)
	info.AddSupportedNIP(19)
	info.AddSupportedNIP(1)
	info.AddSupportedNIP(18)

	for i, v := range []int{0, 1, 2, 12, 13, 17, 18, 19, 44} {
		if info.SupportedNIPs[i] != v {
			t.Errorf("expected info.SupportedNIPs[%d] to equal %v, got %v",
				i, v, info.SupportedNIPs)
			return
		}
	}
}

func TestFetch(t *testing.T) {
	res, err := Fetch(context.Background(), "wss://relay.nostr.bg")
	if err != nil || res.Name == "" {
		t.Errorf("failed to fetch from wss")
	}
	res, err = Fetch(context.Background(), "https://relay.nostr.bg")
	if err != nil || res.Name == "" {
		t.Errorf("failed to fetch from https")
	}
	res, err = Fetch(context.Background(), "relay.nostr.bg")
	if err != nil || res.Name == "" {
		t.Errorf("failed to fetch without protocol")
	}
}
