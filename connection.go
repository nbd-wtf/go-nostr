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

type Connection struct {
	conn *ws.Conn
}

func NewConnection(ctx context.Context, url string, requestHeader http.Header, tlsConfig *tls.Config) (*Connection, error) {
	c, _, err := ws.Dial(ctx, url, getConnectionOptions(requestHeader, tlsConfig))
	if err != nil {
		return nil, err
	}

	c.SetReadLimit(262144) // this should be enough for contact lists of over 2000 people

	return &Connection{
		conn: c,
	}, nil
}

func (c *Connection) WriteMessage(ctx context.Context, data []byte) error {
	if err := c.conn.Write(ctx, ws.MessageText, data); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

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

func (c *Connection) Close() error {
	return c.conn.Close(ws.StatusNormalClosure, "")
}

func (c *Connection) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeoutCause(ctx, time.Millisecond*800, errors.New("ping took too long"))
	defer cancel()
	return c.conn.Ping(ctx)
}
