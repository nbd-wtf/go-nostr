package nostr

import (
	"net/url"
	"strings"
)

// NormalizeURL normalizes the url and replaces http://, https:// schemes by ws://, wss://.
func NormalizeURL(u string) string {
	if u == "" {
		return ""
	}

	u = strings.TrimSpace(u)
	u = strings.ToLower(u)

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
