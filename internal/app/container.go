package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"velocity/internal/config"
	"velocity/internal/engine/recovery"
	"velocity/internal/engine/registry"
	"velocity/internal/engine/wal"
	"velocity/internal/marketdata"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/persistence/postgres/tx"
	"velocity/internal/persistence/worker"
	"velocity/internal/service/orderservice"
	"velocity/internal/transport/http/handler"
	wsHandler "velocity/internal/transport/ws/handler"
	"velocity/internal/userstream"
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

	UserRepository     repository.UserRepository
	OrderRepository    repository.OrderRepository
	TradeRepository    repository.TradeRepository
	PositionRepository repository.PositionRepository
	SymbolRepository   repository.SymbolRepository

	TxManager tx.Manager

	TradeWorker   worker.TradePersistenceWorker
	TradeConsumer *worker.TradeConsumer

	MarketHub *marketdata.Hub
	WSHandler *wsHandler.Handler

	Registry *registry.Registry

	Recovery *recovery.Recovery

	// Future

	//Service
	OrderService *orderservice.Service

	//Handler
	OrderHandler *handler.OrderHandler

	SnapshotRecovery *recovery.SnapshotRecovery

	WALManager      *wal.Manager
	MarketPublisher *marketdata.Publisher
	Dispatcher      *marketdata.Broadcaster

	UserHub        *userstream.Hub
	UserPublisher  *userstream.Publisher
	UserDispatcher *userstream.Dispatcher

	// Engine     *registry.Registry

	// EventBus   eventbus.Bus
	// Redis      *redis.Client
	// Kafka      *kafka.Client
	// Metrics    *metrics.Registry
}
