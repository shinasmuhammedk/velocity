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

type Container struct {

	// --------------------------------------------------
	// Core
	// --------------------------------------------------

	Config *config.Config
	Logger *zap.Logger

	// --------------------------------------------------
	// Infrastructure
	// --------------------------------------------------

	DB *pgxpool.Pool
	HTTP *fiber.App

	// --------------------------------------------------
	// Repositories
	// --------------------------------------------------

	UserRepository     repository.UserRepository
	OrderRepository    repository.OrderRepository
	TradeRepository    repository.TradeRepository
	PositionRepository repository.PositionRepository
	SymbolRepository   repository.SymbolRepository

	// --------------------------------------------------
	// Transactions
	// --------------------------------------------------

	TxManager tx.Manager

	// --------------------------------------------------
	// Workers
	// --------------------------------------------------

	TradeWorker   worker.TradePersistenceWorker
	TradeConsumer *worker.TradeConsumer

	// --------------------------------------------------
	// Market Data
	// --------------------------------------------------

	MarketHub         *marketdata.Hub
	MarketPublisher   *marketdata.Publisher
	MarketBroadcaster *marketdata.Broadcaster

	WSHandler *wsHandler.Handler

	// --------------------------------------------------
	// User Stream
	// --------------------------------------------------

	UserHub        *userstream.Hub
	UserPublisher  *userstream.Publisher
	UserDispatcher *userstream.Dispatcher

	// --------------------------------------------------
	// Matching Engine
	// --------------------------------------------------

	Registry           *registry.Registry
	Recovery           *recovery.Recovery
	SnapshotRecovery   *recovery.SnapshotRecovery
	WALManager         *wal.Manager

	// --------------------------------------------------
	// Services
	// --------------------------------------------------

	OrderService *orderservice.Service

	// --------------------------------------------------
	// HTTP Handlers
	// --------------------------------------------------

	OrderHandler *handler.OrderHandler
}