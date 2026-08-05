package marketdata

import (

	"github.com/gofiber/contrib/websocket"
)

type Client struct {
    Conn *websocket.Conn
}

// func (c *Client) Send(v any) {
//     fmt.Println("WRITING TO WS")

//     if err := c.Conn.WriteJSON(v); err != nil {
//         fmt.Println("WRITE ERROR:", err)
//         return
//     }

//     fmt.Println("WRITE SUCCESS")
// }

func (c *Client) Send(v any) error {
    return c.Conn.WriteJSON(v)
}