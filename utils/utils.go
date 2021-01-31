package nostrutils

import (
	"net/url"
	"strings"
)

func NormalizeURL(u string) string {
	if !strings.HasPrefix(u, "http") && !strings.HasPrefix(u, "ws") {
		u = "ws://" + u
	}
	p, err := url.Parse(u)
	if err != nil {
		return ""
	}

	if strings.HasSuffix(p.RawPath, "/") {
		p.RawPath = p.RawPath[0 : len(p.RawPath)-1]
	}

	if strings.HasSuffix(p.RawPath, "/ws") {
		p.RawPath = p.RawPath[0 : len(p.RawPath)-3]
	}

	return p.String()
}

func WebsocketURL(u string) string {
	p, _ := url.Parse(NormalizeURL(u))
	p.RawPath += "/ws"
	return p.String()
}
