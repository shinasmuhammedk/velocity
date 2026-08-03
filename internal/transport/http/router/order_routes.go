package router

import (
	"velocity/internal/transport/http/handler"

	"github.com/gofiber/fiber/v2"
)

func RegisterOrderRoutes(
	api fiber.Router,
	orderHandler *handler.OrderHandler,
) {
	api.Post("/orders", orderHandler.Submit)
	api.Get("/orders/:id", orderHandler.GetByID)
	api.Delete("/orders/:id", orderHandler.Cancel)
	api.Patch("/orders/:id", orderHandler.Modify)
	api.Get("/orders/open/:userID", orderHandler.GetOpenOrders)
	api.Get("/orders/history/:userID", orderHandler.OrderHistory)
}
