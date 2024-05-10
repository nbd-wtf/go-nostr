package nip54

import (
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func NormalizeIdentifier(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	res, _, _ := transform.Bytes(norm.NFKC, []byte(name))
	runes := []rune(string(res))

	b := make([]rune, len(runes))
	for i, letter := range runes {
		if unicode.IsLetter(letter) || unicode.IsNumber(letter) {
			b[i] = letter
		} else {
			b[i] = '-'
		}
	}

	return string(b)
}
