package router

import (
	"velocity/internal/transport/http/handler"
	"velocity/internal/transport/http/middleware"

	"github.com/gofiber/fiber/v2"
)

func RegisterOrderRoutes(
	api fiber.Router,
	orderHandler *handler.OrderHandler,
	rateLimit *middleware.RateLimitMiddleware,
) {
	orders := api.Group("/orders")

	orders.Post("", rateLimit.Submit, orderHandler.Submit)

	orders.Get("/open", orderHandler.GetOpenOrders)
	orders.Get("/history", orderHandler.OrderHistory)

	orders.Get("/:id", orderHandler.GetByID)
	orders.Delete("/:id", rateLimit.Cancel, orderHandler.Cancel)
	orders.Patch("/:id", rateLimit.Modify, orderHandler.Modify)
}
