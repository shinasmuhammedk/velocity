package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"velocity/internal/infrastructure/metrics"
)

func Metrics() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Prometheus scraping should not be counted as an API request.
		if c.Path() == "/metrics" {
			return c.Next()
		}

		start := time.Now()

		err := c.Next()

		duration := time.Since(start)

		route := c.Route().Path
		if route == "" {
			route = "unknown"
		}

		status := strconv.Itoa(c.Response().StatusCode())
		method := c.Method()

		metrics.HTTPRequestsTotal.
			WithLabelValues(method, route, status).
			Inc()

		metrics.HTTPRequestDuration.
			WithLabelValues(method, route, status).
			Observe(duration.Seconds())

		return err
	}
}
