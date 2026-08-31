package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"velocity/internal/infrastructure/kafka"
	"velocity/internal/infrastructure/redis"
	"velocity/pkg/response"
)

type HealthHandler struct {
	db          *pgxpool.Pool
	redisHealth *redis.HealthChecker
	kafkaHealth *kafka.HealthChecker
}

func NewHealthHandler(
	db *pgxpool.Pool,
	redisHealth *redis.HealthChecker,
	kafkaHealth *kafka.HealthChecker,
) *HealthHandler {
	return &HealthHandler{
		db:          db,
		redisHealth: redisHealth,
		kafkaHealth: kafkaHealth,
	}
}

// Live reports whether the process itself is running.
//
// It intentionally does not check dependencies.
func (h *HealthHandler) Live(c *fiber.Ctx) error {
	return response.Success(
		c,
		fiber.StatusOK,
		"ok",
		fiber.Map{
			"status": "up",
		},
	)
}

// Ready reports whether the process can currently serve traffic.
//
// Readiness checks all critical downstream dependencies:
// PostgreSQL, Redis, and Kafka.
//
// Kafka readiness additionally verifies that the required
// Velocity topics exist and contain partitions.
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(
		c.Context(),
		2*time.Second,
	)
	defer cancel()

	checks := fiber.Map{}
	allHealthy := true

	// -------------------------
	// PostgreSQL
	// -------------------------

	if err := h.db.Ping(ctx); err != nil {
		checks["database"] = fiber.Map{
			"status": "down",
			"error":  err.Error(),
		}

		allHealthy = false
	} else {
		checks["database"] = fiber.Map{
			"status": "up",
		}
	}

	// -------------------------
	// Redis
	// -------------------------

	if err := h.redisHealth.Check(ctx); err != nil {
		checks["redis"] = fiber.Map{
			"status": "down",
			"error":  err.Error(),
		}

		allHealthy = false
	} else {
		checks["redis"] = fiber.Map{
			"status": "up",
		}
	}

	// -------------------------
	// Kafka
	// -------------------------

	if err := h.kafkaHealth.Readiness(ctx); err != nil {
		checks["kafka"] = fiber.Map{
			"status": "down",
			"error":  err.Error(),
		}

		allHealthy = false
	} else {
		checks["kafka"] = fiber.Map{
			"status": "up",
		}
	}

	if !allHealthy {
		return response.Success(
			c,
			fiber.StatusServiceUnavailable,
			"not ready",
			checks,
		)
	}

	return response.Success(
		c,
		fiber.StatusOK,
		"ready",
		checks,
	)
}
