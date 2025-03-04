package blossom

import (
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/valyala/fasthttp"
)

// Client represents a Blossom client for interacting with a media server
type Client struct {
	mediaserver string
	httpClient  *fasthttp.Client
	signer      nostr.Signer
}

// NewClient creates a new Blossom client
func NewClient(mediaserver string, signer nostr.Signer) *Client {
	if !strings.HasPrefix(mediaserver, "http") {
		mediaserver = "https://" + mediaserver
	}

	return &Client{
		mediaserver: strings.TrimSuffix(mediaserver, "/") + "/",
		httpClient:  createHTTPClient(),
		signer:      signer,
	}
}

// createHTTPClient creates a properly configured HTTP client
func createHTTPClient() *fasthttp.Client {
	readTimeout, _ := time.ParseDuration("10s")
	writeTimeout, _ := time.ParseDuration("10s")
	maxIdleConnDuration, _ := time.ParseDuration("1h")
	return &fasthttp.Client{
		ReadTimeout:                   readTimeout,
		WriteTimeout:                  writeTimeout,
		MaxIdleConnDuration:           maxIdleConnDuration,
		NoDefaultUserAgentHeader:      true, // Don't send: User-Agent: fasthttp
		DisableHeaderNamesNormalizing: true, // If you set the case on your headers correctly you can enable this
		DisablePathNormalizing:        true,
		// increase DNS cache time to an hour instead of default minute
		Dial: (&fasthttp.TCPDialer{
			Concurrency:      4096,
			DNSCacheDuration: time.Hour,
		}).Dial,
	}
}

// GetSigner returns the client's signer
func (c *Client) GetSigner() nostr.Signer {
	return c.signer
}

// GetMediaServer returns the client's media server URL
func (c *Client) GetMediaServer() string {
	return c.mediaserver
}
