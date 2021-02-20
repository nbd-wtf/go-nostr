package nostrutils

import (
	"net/url"
	"strings"
)

func NormalizeURL(u string) string {
	if !strings.HasPrefix(u, "http") && !strings.HasPrefix(u, "ws") {
		u = "wss://" + u
	}
	p, err := url.Parse(u)
	if err != nil {
		return ""
	}

	if p.Scheme == "http" {
		p.Scheme = "ws"
	} else if p.Scheme == "https" {
		p.Scheme = "wss"
	}

	if strings.HasSuffix(p.RawPath, "/") {
		p.RawPath = p.RawPath[0 : len(p.RawPath)-1]
	}

	return p.String()
}
