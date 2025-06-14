package nip11

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
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

	assert.Contains(t, info.SupportedNIPs, 0, 1, 2, 12, 13, 17, 18, 19, 44)
}

func TestAddSupportedNIPs(t *testing.T) {
	info := RelayInformationDocument{}
	info.AddSupportedNIPs([]int{0, 1, 2, 12, 13, 17, 18, 19, 44})

	assert.Contains(t, info.SupportedNIPs, 0, 1, 2, 12, 13, 17, 18, 19, 44)
}

func TestFetch(t *testing.T) {
	tests := []struct {
		inputURL     string
		expectError  bool
		expectedName string
		expectedURL  string
	}{
		{"wss://nostr.wine", false, "", "wss://nostr.wine"},
		{"https://nostr.wine", false, "", "wss://nostr.wine"},
		{"nostr.wine", false, "", "wss://nostr.wine"},
		{"nos.lol", false, "", "wss://nos.lol"},
		{"https://nos.lol", false, "", "wss://nos.lol"},
		{"wss://nos.lol", false, "", "wss://nos.lol"},
		{"wlenwqkeqwe.asjdaskd", true, "", "wss://wlenwqkeqwe.asjdaskd"},
	}

	for _, test := range tests {
		res, err := Fetch(context.Background(), test.inputURL)

		if test.expectError {
			assert.Error(t, err, "expected error for URL: %s", test.inputURL)
			assert.NotNil(t, res, "expected result not to be nil for URL: %s", test.inputURL)
			assert.Equal(t, test.expectedURL, res.URL, "expected URL to be %s for input: %s", test.expectedURL, test.inputURL)
		} else {
			assert.Nil(t, err, "unexpect error for URL: %s", test.inputURL)
			assert.NotEmpty(t, res.Name, "expected non-empty name for URL: %s", test.inputURL)
			assert.Equal(t, test.expectedURL, res.URL, "expected URL to be %s for input: %s", test.expectedURL, test.inputURL)
		}
	}
}
