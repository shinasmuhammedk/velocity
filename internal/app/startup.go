package app

import (
	"context"
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"velocity/internal/config"
	"velocity/internal/infrastructure/redis"
	"velocity/internal/persistence/postgres"
	"velocity/internal/transport/http/middleware"
	"velocity/pkg/logger"
)

// allowedOrigin returns the origin(s) the API accepts browser requests
// from. This only matters when the frontend calls the API directly
// (e.g. a production build not served through the Vite dev proxy).
// Override with the FRONTEND_ORIGIN env var (comma-separated for
// multiple origins) when deploying somewhere other than localhost:5173.
func allowedOrigin() string {
	if origin := os.Getenv("FRONTEND_ORIGIN"); origin != "" {
		return origin
	}
	return "http://localhost:5173"
}

// Startup initializes all application dependencies
// and returns a fully populated container.
func Startup() (*Container, error) {

	container := &Container{}

	// Configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	container.Config = cfg

	// Logger
	if err := logger.Init(cfg.App.Environment); err != nil {
		return nil, fmt.Errorf(
			"initialize logger: %w",
			err,
		)
	}

	container.Logger = logger.Logger()

	container.Logger.Info(
		"configuration loaded successfully",
	)

	// Database
	db, err := postgres.New(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf(
			"initialize postgres: %w",
			err,
		)
	}

	container.DB = db

	container.Logger.Info(
		"postgres connection established",
	)

	//redis
	redisClient := redis.New(cfg.Redis)
	if err := redisClient.Ping(context.Background()); err != nil {
		_ = redisClient.Close()

		return nil, fmt.Errorf(
			"initialize redis: %w",
			err,
		)
	}
	container.Redis = redisClient
	container.Logger.Info("redis connection established")

	// HTTP Server
	container.HTTP = fiber.New()

	container.HTTP.Use(cors.New(cors.Config{
		// Vite dev server origin + configurable via FRONTEND_ORIGIN for other environments.
		AllowOrigins:     allowedOrigin(),
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-User-Id",
		AllowMethods:     "GET, POST, PATCH, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))
	container.HTTP.Use(middleware.Metrics())
	container.HTTP.Use(recover.New())

	container.HTTP.Use(func(c *fiber.Ctx) error {
		return c.Next()
	})

	container.HTTP.Use(recover.New())

	return container, nil
}