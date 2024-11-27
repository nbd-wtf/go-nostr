package nip05

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input          string
		expectedName   string
		expectedDomain string
		expectError    bool
	}{
		{"saknd@yyq.com", "saknd", "yyq.com", false},
		{"287354gkj+asbdfo8gw3rlicbsopifbcp3iougb5piseubfdikswub5ks@yyq.com", "287354gkj+asbdfo8gw3rlicbsopifbcp3iougb5piseubfdikswub5ks", "yyq.com", false},
		{"asdn.com", "_", "asdn.com", false},
		{"_@uxux.com.br", "_", "uxux.com.br", false},
		{"821yh498ig21", "", "", true},
		{"////", "", "", true},
	}

	for _, test := range tests {
		name, domain, err := ParseIdentifier(test.input)
		if test.expectError {
			assert.Error(t, err, "expected error for input: %s", test.input)
		} else {
			assert.NoError(t, err, "not expect error for input: %s", test.input)
			assert.Equal(t, test.expectedName, name)
			assert.Equal(t, test.expectedDomain, domain)
		}
	}
}

func TestQuery(t *testing.T) {
	tests := []struct {
		input       string
		expectedKey string
		expectError bool
	}{
		{"fiatjaf.com", "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d", false},
		{"htlc@fiatjaf.com", "f9dd6a762506260b38a2d3e5b464213c2e47fa3877429fe9ee60e071a31a07d7", false},
	}

	for _, test := range tests {
		pp, err := QueryIdentifier(context.Background(), test.input)
		if test.expectError {
			assert.Error(t, err, "expected error for input: %s", test.input)
		} else {
			assert.NoError(t, err, "did not expect error for input: %s", test.input)
			assert.Equal(t, test.expectedKey, pp.PublicKey, "for input: %s", test.input)
		}
	}
}
