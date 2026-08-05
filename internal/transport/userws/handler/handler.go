package handler

import (
	"strconv"
	"velocity/internal/userstream"

	"github.com/gofiber/contrib/websocket"
)

type Handler struct {
	hub *userstream.Hub
}

func NewHandler(
	hub *userstream.Hub,
) *Handler {
	return &Handler{
		hub: hub,
	}
}

func (h *Handler) Handle(
	c *websocket.Conn,
) {

	userID, err := strconv.ParseInt(
		c.Query("user_id"),
		10,
		64,
	)

	if err != nil {
		_ = c.Close()
		return
	}


	client := &userstream.Client{
		Conn: c,
	}


	h.hub.Subscribe(
		userID,
		client,
	)


	defer func() {

		h.hub.Unsubscribe(
			userID,
			client,
		)

		client.Close()

	}()


	for {

		if _, _, err := c.ReadMessage(); err != nil {
			break
		}

	}
}