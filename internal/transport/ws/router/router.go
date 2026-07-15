package router

import (
	"velocity/internal/transport/ws/handler"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func Register(
	app *fiber.App,
	handler *handler.Handler,
) {

	app.Use(
		"/ws/:symbol",
		func(c *fiber.Ctx) error {
			if websocket.IsWebSocketUpgrade(c) {
				return c.Next()
			}

			return fiber.ErrUpgradeRequired
		},
	)

	app.Get(
		"/ws/:symbol",
		websocket.New(handler.Handle),
	)
}