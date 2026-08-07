package handler

import (
	"velocity/internal/transport/http/middleware"
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

func (h *Handler) Handle(c *websocket.Conn) {

	user, ok := c.Locals("authUser").(*middleware.AuthenticatedUser)

	if !ok || user == nil || user.UserID == 0 {
		_ = c.Close()
		return
	}

	userID := user.UserID

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