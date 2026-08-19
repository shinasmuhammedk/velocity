package router

import (
	"github.com/gofiber/fiber/v2"

	"velocity/internal/transport/http/handler"
)

func RegisterHealthRoutes(
	app fiber.Router,
	healthHandler *handler.HealthHandler,
) {
	health := app.Group("/health")

	health.Get("/", healthHandler.Live)
	health.Get("/ready", healthHandler.Ready)
}