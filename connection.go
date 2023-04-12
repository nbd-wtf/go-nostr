package nostr

import (
	"sync"

	"github.com/gorilla/websocket"
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

func (c *Connection) WriteJSON(v any) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.socket.WriteJSON(v)
}

func (c *Connection) WriteMessage(messageType int, data []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.socket.WriteMessage(messageType, data)
}

func (c *Connection) Close() error {
	return c.socket.Close()
}
