package nip54

import (
	"regexp"
	"strings"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var nonLetter = regexp.MustCompile(`\W`)

func NormalizeIdentifier(name string) string {
	res, _, _ := transform.Bytes(norm.NFKC, []byte(name))
	str := nonLetter.ReplaceAllString(string(res), "-")
	return strings.ToLower(str)
}
