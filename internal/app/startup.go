package app

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"velocity/internal/config"
	"velocity/internal/persistence/postgres"
	"velocity/pkg/logger"
)

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

	// HTTP Server
	container.HTTP = fiber.New()
	container.HTTP.Use(recover.New())

	return container, nil
}
