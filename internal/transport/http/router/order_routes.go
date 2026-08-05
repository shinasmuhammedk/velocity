package router

import (
	"velocity/internal/transport/http/handler"

	"github.com/gofiber/fiber/v2"
)

func RegisterOrderRoutes(
	api fiber.Router,
	orderHandler *handler.OrderHandler,
) {
	orders := api.Group("/orders")

	orders.Post("/", orderHandler.Submit)

	orders.Get("/:id", orderHandler.GetByID)

	orders.Delete("/:id", orderHandler.Cancel)

	orders.Patch("/:id", orderHandler.Modify)

	orders.Get("/open", orderHandler.GetOpenOrders)

	orders.Get("/history", orderHandler.OrderHistory)
}
