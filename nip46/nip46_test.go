package nip46

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidBunkerURL(t *testing.T) {
	valid := IsValidBunkerURL("bunker://3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d?relay=wss%3A%2F%2Frelay.damus.io&relay=wss%3A%2F%2Frelay.snort.social&relay=wss%3A%2F%2Frelay.nsecbunker.com")
	assert.True(t, valid, "should be valid")

	inValid := IsValidBunkerURL("askjdbkajdbv")
	assert.False(t, inValid, "should be invalid")

	inValid1 := IsValidBunkerURL("asdjasbndksa@asjdnksa.com")
	assert.False(t, inValid1, "should be invalid")

	inValid2 := IsValidBunkerURL("https://hello.com?relays=wss://xxxxxx.xxxx")
	assert.False(t, inValid2, "should be invalid")

	inValid3 := IsValidBunkerURL("bunker://fa883d107ef9e558472c4eb9aaaefa459d?relay=wss%3A%2F%2Frelay.damus.io&relay=wss%3A%2F%2Frelay.snort.social&relay=wss%3A%2F%2Frelay.nsecbunker.com")
	assert.False(t, inValid3, "should be invalid")
}
