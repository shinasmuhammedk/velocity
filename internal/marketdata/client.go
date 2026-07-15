package marketdata

import "github.com/gofiber/contrib/websocket"

type Client struct {
    Conn *websocket.Conn
}

func (c *Client) Send(v any) {
    _ = c.Conn.WriteJSON(v)
}