package app

import (
	"fmt"

	"velocity/internal/config"
	"velocity/pkg/logger"
)

// Startup initializes all application dependencies
// and returns a fully populated container.
func Startup() (*Container, error) {

	container := &Container{}

	// --------------------------------------------------
	// Load Configuration
	// --------------------------------------------------

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	container.Config = cfg

	// --------------------------------------------------
	// Initialize Logger
	// --------------------------------------------------

	if err := logger.Init(cfg.App.Environment); err != nil {
		return nil, fmt.Errorf("initialize logger: %w", err)
	}

	container.Logger = logger.Logger()

	container.Logger.Info("configuration loaded successfully")

	// --------------------------------------------------
	// Database
	// --------------------------------------------------
	// TODO:
	// container.DB = postgres.New(cfg.Database)

	// --------------------------------------------------
	// HTTP Server
	// --------------------------------------------------
	// TODO:
	// container.HTTP = server.New(cfg.Server)

	return container, nil
}