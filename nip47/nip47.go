package nip47

import (
	"errors"
	"net/url"

	"github.com/nbd-wtf/go-nostr"
)

type NWCURIParts struct {
	clientSecretKey string
	walletPublicKey string
	relays          []string
}

// extracts the NWC URI parts from a connection URI
func ParseNWCURI(nwcUri string) (*NWCURIParts, error) {
	p, err := url.Parse(nwcUri)
	if err != nil {
		return nil, err
	}
	if p.Scheme != "nostr+walletconnect" {
		return nil, errors.New("incorrect scheme")
	}
	if !nostr.IsValid32ByteHex(p.Host) {
		return nil, errors.New("invalid wallet public key")
	}
	query := p.Query()
	relays := query["relay"]
	secret := query.Get("secret")
	if !nostr.IsValid32ByteHex(secret) {
		return nil, errors.New("invalid secret")
	}
	if len(relays) == 0 {
		return nil, errors.New("no relays")
	}

	return &NWCURIParts{
		walletPublicKey: p.Host,
		clientSecretKey: secret,
		relays:          relays,
	}, nil
}
