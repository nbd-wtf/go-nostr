package nostr

import (
	"crypto/tls"
	"net/http"

	ws "github.com/coder/websocket"
)

var emptyOptions = ws.DialOptions{}

func getConnectionOptions(requestHeader http.Header, tlsConfig *tls.Config) *ws.DialOptions {
	// on javascript we ignore everything because there is nothing else we can do
	return &emptyOptions
}
