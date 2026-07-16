package router

import (
	"velocity/internal/transport/http/handler"

	"github.com/gofiber/fiber/v2"
)

func Register(
	app *fiber.App,
	orderHandler *handler.OrderHandler,
) {
	api := app.Group("/api")

	api.Post("/orders", orderHandler.Submit)
    api.Delete("/orders/:id", orderHandler.Cancel)
    api.Patch("/orders/:id", orderHandler.Modify)
}
