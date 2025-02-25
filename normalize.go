package nostr

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ImVexed/fasturl"
)

// NormalizeURL normalizes the url and replaces http://, https:// schemes with ws://, wss://
// and normalizes the path.
func NormalizeURL(u string) string {
	if u == "" {
		return ""
	}

	u = strings.TrimSpace(u)
	p, err := fasturl.ParseURL(u)
	if err != nil {
		return ""
	}

	// the fabulous case of localhost:1234 that considers "localhost" the protocol and "123" the host
	if p.Port == "" && len(p.Protocol) > 5 {
		p.Protocol, p.Host, p.Port = "", p.Protocol, p.Host
	}

	if p.Protocol == "" {
		if p.Host == "localhost" || p.Host == "127.0.0.1" {
			p.Protocol = "ws"
		} else {
			p.Protocol = "wss"
		}
	} else if p.Protocol == "https" {
		p.Protocol = "wss"
	} else if p.Protocol == "http" {
		p.Protocol = "ws"
	}

	p.Host = strings.ToLower(p.Host)
	p.Path = strings.TrimRight(p.Path, "/")

	var buf strings.Builder
	buf.Grow(
		len(p.Protocol) + 3 + len(p.Host) + 1 + len(p.Port) + len(p.Path) + 1 + len(p.Query),
	)

	buf.WriteString(p.Protocol)
	buf.WriteString("://")
	buf.WriteString(p.Host)
	if p.Port != "" {
		buf.WriteByte(':')
		buf.WriteString(p.Port)
	}
	buf.WriteString(p.Path)
	if p.Query != "" {
		buf.WriteByte('?')
		buf.WriteString(p.Query)
	}
	return buf.String()
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
