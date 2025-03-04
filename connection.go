package nostr

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	ws "github.com/coder/websocket"
)

// Connection represents a websocket connection to a Nostr relay.
type Connection struct {
	conn *ws.Conn
}

// NewConnection creates a new websocket connection to a Nostr relay.
func NewConnection(ctx context.Context, url string, requestHeader http.Header, tlsConfig *tls.Config) (*Connection, error) {
	c, _, err := ws.Dial(ctx, url, getConnectionOptions(requestHeader, tlsConfig))
	if err != nil {
		return nil, err
	}

	c.SetReadLimit(2 << 24) // 33MB

	return &Connection{
		conn: c,
	}, nil
}

// WriteMessage writes arbitrary bytes to the websocket connection.
func (c *Connection) WriteMessage(ctx context.Context, data []byte) error {
	if err := c.conn.Write(ctx, ws.MessageText, data); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// ReadMessage reads arbitrary bytes from the websocket connection into the provided buffer.
func (c *Connection) ReadMessage(ctx context.Context, buf io.Writer) error {
	_, reader, err := c.conn.Reader(ctx)
	if err != nil {
		return fmt.Errorf("failed to get reader: %w", err)
	}
	if _, err := io.Copy(buf, reader); err != nil {
		return fmt.Errorf("failed to read message: %w", err)
	}
	return nil
}

// Close closes the websocket connection.
func (c *Connection) Close() error {
	return c.conn.Close(ws.StatusNormalClosure, "")
}

// Ping sends a ping message to the websocket connection.
func (c *Connection) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeoutCause(ctx, time.Millisecond*800, errors.New("ping took too long"))
	defer cancel()
	return c.conn.Ping(ctx)
}
