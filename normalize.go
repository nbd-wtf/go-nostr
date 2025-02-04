package nostr

import (
	"fmt"
	"net/url"
	"strings"
)

// NormalizeURL normalizes the url and replaces http://, https:// schemes with ws://, wss://
// and normalizes the path.
func NormalizeURL(u string) string {
	if u == "" {
		return ""
	}

	u = strings.TrimSpace(u)
	if fqn := strings.Split(u, ":")[0]; fqn == "localhost" || fqn == "127.0.0.1" {
		u = "ws://" + u
	} else if !strings.HasPrefix(u, "http") && !strings.HasPrefix(u, "ws") {
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

	p.Host = strings.ToLower(p.Host)
	p.Path = strings.TrimRight(p.Path, "/")

	return p.String()
}

// NormalizeHTTPURL does normalization of http(s):// URLs according to rfc3986. Don't use for relay URLs.
func NormalizeHTTPURL(s string) (string, error) {
	s = strings.TrimSpace(s)

	if !strings.HasPrefix(s, "http") {
		s = "https://" + s
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	if u.Scheme == "" {
		u, err = url.Parse("http://" + s)
		if err != nil {
			return s, err
		}
	}

	if strings.HasPrefix(s, "//") {
		s = "http:" + s
	}

	var p int
	switch u.Scheme {
	case "http":
		p = 80
	case "https":
		p = 443
	}
	u.Host = strings.TrimSuffix(u.Host, fmt.Sprintf(":%d", p))

	v := u.Query()
	u.RawQuery = v.Encode()
	u.RawQuery, _ = url.QueryUnescape(u.RawQuery)

	h := u.String()
	h = strings.TrimSuffix(h, "/")

	return h, nil
}

// NormalizeOKMessage takes a string message that is to be sent in an `OK` or `CLOSED` command
// and prefixes it with "<prefix>: " if it doesn't already have an acceptable prefix.
func NormalizeOKMessage(reason string, prefix string) string {
	if idx := strings.Index(reason, ": "); idx == -1 || strings.IndexByte(reason[0:idx], ' ') != -1 {
		return prefix + ": " + reason
	}
	return reason
}
