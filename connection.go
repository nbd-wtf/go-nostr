package nostr

import (
	"context"
	json "encoding/json"
	"sync"

	"nhooyr.io/websocket"
)

type Connection struct {
	socket *websocket.Conn
	mutex  sync.Mutex
}

func NewConnection(socket *websocket.Conn) *Connection {
	return &Connection{
		socket: socket,
	}
}

func (c *Connection) WriteJSON(ctx context.Context, v any) (rerr error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	w, err := c.socket.Writer(ctx, websocket.MessageText)
	if err != nil {
		return err
	}
	defer func() {
		cerr := w.Close()
		if rerr == nil {
			rerr = cerr
		}
	}()
	return json.NewEncoder(w).Encode(v)
}

func (c *Connection) WriteMessage(ctx context.Context, messageType websocket.MessageType, data []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.socket.Write(ctx, messageType, data)
}

func (c *Connection) Close() error {
	return c.socket.Close(websocket.StatusNormalClosure, "")
}
