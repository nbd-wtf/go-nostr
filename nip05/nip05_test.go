package nip05

import (
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
