package router

import (
	"velocity/internal/transport/http/middleware"
	"velocity/internal/transport/userws/handler"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func Register(
	app *fiber.App,
	handler *handler.Handler,
	auth *middleware.AuthMiddleware,
) {

	app.Use(
		"/ws/private",
		func(c *fiber.Ctx) error {

			if !websocket.IsWebSocketUpgrade(c) {
				return fiber.ErrUpgradeRequired
			}

			return c.Next()
		},
	)

	app.Use(
		"/ws/private",
		auth.AuthenticateWS,
	)

	app.Get(
		"/ws/private",
		websocket.New(handler.Handle),
	)
}
