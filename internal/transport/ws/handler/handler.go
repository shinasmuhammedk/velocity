package handler

import (
	"velocity/internal/marketdata"

	"github.com/gofiber/contrib/websocket"
)

type Handler struct {
	hub *marketdata.Hub
}

func NewHandler(
	hub *marketdata.Hub,
) *Handler {
	return &Handler{
		hub: hub,
	}
}

func (h *Handler) Handle(
	c *websocket.Conn,
) {
	symbol := c.Params("symbol")

	client := &marketdata.Client{
		Conn: c,
	}

	h.hub.Subscribe(
		symbol,
		client,
	)

	defer h.hub.Unsubscribe(
		symbol,
		client,
	)

	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			break
		}
	}
}