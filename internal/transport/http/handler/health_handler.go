package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"velocity/internal/infrastructure/redis"
	"velocity/pkg/response"
)

type HealthHandler struct {
	db          *pgxpool.Pool
	redisHealth *redis.HealthChecker
}

func NewHealthHandler(
	db *pgxpool.Pool,
	redisHealth *redis.HealthChecker,
) *HealthHandler {

	return &HealthHandler{
		db:          db,
		redisHealth: redisHealth,
	}
}

// Live reports whether the process itself is up.
//
// It intentionally does not check any dependency (DB, Redis, Kafka).
// This is what an orchestrator should call to decide whether to
// restart the container - a dependency being down should not cause
// a healthy process to be killed.
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
// Unlike Live, this checks downstream dependencies. It's meant for
// load balancers / k8s readiness probes deciding whether to route
// traffic to this instance.
func (h *HealthHandler) Ready(c *fiber.Ctx) error {

	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()

	checks := fiber.Map{}
	allHealthy := true

	// -------------------------
	// Postgres
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
	//
	// Not checked yet: Producer has no ping/dial-check method today.
	// Marked "unknown" rather than silently omitted, so this endpoint
	// doesn't imply a guarantee it isn't making.

	checks["kafka"] = fiber.Map{
		"status": "unknown",
		"note":   "no connectivity check implemented",
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