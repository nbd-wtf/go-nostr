package nostr

import (
	"net/url"
	"strings"
)

func NormalizeURL(u string) string {
	if u == "" {
		return ""
	}

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

	p.Path = strings.TrimRight(p.Path, "/")

	return p.String()
}
