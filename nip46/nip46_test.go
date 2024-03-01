package nip46

import "testing"

func TestValidBunkerURL(t *testing.T) {
	if !IsValidBunkerURL("bunker://3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d?relay=wss%3A%2F%2Frelay.damus.io&relay=wss%3A%2F%2Frelay.snort.social&relay=wss%3A%2F%2Frelay.nsecbunker.com") {
		t.Fatalf("should be valid")
	}
	if IsValidBunkerURL("askjdbkajdbv") {
		t.Fatalf("should be invalid")
	}
	if IsValidBunkerURL("asdjasbndksa@asjdnksa.com") {
		t.Fatalf("should be invalid")
	}
	if IsValidBunkerURL("https://hello.com?relays=wss://xxxxxx.xxxx") {
		t.Fatalf("should be invalid")
	}
	if IsValidBunkerURL("bunker://fa883d107ef9e558472c4eb9aaaefa459d?relay=wss%3A%2F%2Frelay.damus.io&relay=wss%3A%2F%2Frelay.snort.social&relay=wss%3A%2F%2Frelay.nsecbunker.com") {
		t.Fatalf("should be invalid")
	}
}
