package handler

import (
	"velocity/internal/marketdata"
	"velocity/pkg/constants"

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

	client := &marketdata.Client{
		Conn: c,
	}

	for {

		var req marketdata.ClientRequest

		if err := c.ReadJSON(&req); err != nil {
			break
		}

		switch req.Action {

		case constants.ActionSubscribe:

			h.hub.Subscribe(
				req.Symbol,
				client,
			)

			client.Send(
				marketdata.ServerResponse{
					Type: constants.ResponseSubscribed,
					Data: req.Symbol,
				},
			)

		case constants.ActionUnsubscribe:

			h.hub.Unsubscribe(
				req.Symbol,
				client,
			)

			client.Send(
				marketdata.ServerResponse{
					Type: constants.ResponseUnsubscribed,
					Data: req.Symbol,
				},
			)

		case constants.ActionPing:

			client.Send(
				marketdata.ServerResponse{
					Type: constants.ResponsePong,
				},
			)
		}
	}
}
