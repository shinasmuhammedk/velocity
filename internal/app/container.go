package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"velocity/internal/config"
	"velocity/internal/persistence/postgres/repository"
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
	DB *pgxpool.Pool

	// Transport
	HTTP *fiber.App
    
    UserRepository repository.UserRepository

	// Future
	//
	// Engine     *registry.Registry
	// EventBus   eventbus.Bus
	// Redis      *redis.Client
	// Kafka      *kafka.Client
	// Metrics    *metrics.Registry
}