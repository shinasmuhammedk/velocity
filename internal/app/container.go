package app

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"velocity/internal/config"
)

// Container holds all shared application dependencies.
//
// It acts as the application's dependency injection container.
// Every subsystem is initialized once during startup
// and stored here for reuse throughout the application.
type Container struct {

	// Core
	Config *config.Config
	Logger *zap.Logger

	// Infrastructure
	DB *sql.DB

	// Transport
	HTTP *fiber.App

	// Future
	//
	// Engine     *registry.Registry
	// EventBus   eventbus.Bus
	// Redis      *redis.Client
	// Kafka      *kafka.Client
	// Metrics    *metrics.Registry
}