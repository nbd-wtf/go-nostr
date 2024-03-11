package nip05

import (
	"context"
	"testing"
)

func TestParse(t *testing.T) {
	name, domain, _ := ParseIdentifier("saknd@yyq.com")
	if name != "saknd" || domain != "yyq.com" {
		t.Fatalf("wrong parsing")
	}

	name, domain, _ = ParseIdentifier("287354gkj+asbdfo8gw3rlicbsopifbcp3iougb5piseubfdikswub5ks@yyq.com")
	if name != "287354gkj+asbdfo8gw3rlicbsopifbcp3iougb5piseubfdikswub5ks" || domain != "yyq.com" {
		t.Fatalf("wrong parsing")
	}

	name, domain, _ = ParseIdentifier("asdn.com")
	if name != "_" || domain != "asdn.com" {
		t.Fatalf("wrong parsing")
	}

	name, domain, _ = ParseIdentifier("_@uxux.com.br")
	if name != "_" || domain != "uxux.com.br" {
		t.Fatalf("wrong parsing")
	}

	_, _, err := ParseIdentifier("821yh498ig21")
	if err == nil {
		t.Fatalf("should have errored")
	}

	_, _, err = ParseIdentifier("////")
	if err == nil {
		t.Fatalf("should have errored")
	}
}

func TestQuery(t *testing.T) {
	pp, err := QueryIdentifier(context.Background(), "fiatjaf.com")
	if err != nil || pp.PublicKey != "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d" {
		t.Fatalf("invalid query for fiatjaf.com")
	}

	pp, err = QueryIdentifier(context.Background(), "htlc@fiatjaf.com")
	if err != nil || pp.PublicKey != "f9dd6a762506260b38a2d3e5b464213c2e47fa3877429fe9ee60e071a31a07d7" {
		t.Fatalf("invalid query for htlc@fiatjaf.com")
	}
}
