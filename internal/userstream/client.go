package userstream

import (
	"sync"

	"github.com/gofiber/contrib/websocket"
)

type Client struct {
	Conn *websocket.Conn
	mu   sync.Mutex
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		Conn: conn,
	}
}

func (c *Client) Send(message any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.Conn.WriteJSON(message)
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.Conn.Close()
}