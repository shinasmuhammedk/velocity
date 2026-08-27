package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"velocity/internal/analytics/candles"
	"velocity/internal/analytics/stats"
	"velocity/internal/config"
	"velocity/internal/engine/recovery"
	"velocity/internal/engine/registry"
	"velocity/internal/engine/wal"
	"velocity/internal/marketdata"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/persistence/postgres/tx"
	"velocity/internal/persistence/worker"
	"velocity/internal/service/marketservice"
	"velocity/internal/service/orderservice"
	"velocity/internal/service/positionservice"
	"velocity/internal/service/riskservice"
	"velocity/internal/service/settlementservice"
	"velocity/internal/service/userservice"
	"velocity/internal/service/walletservice"
	identityclient "velocity/internal/transport/grpc/client/identity"
	grpcserver "velocity/internal/transport/grpc/server"
	"velocity/internal/transport/http/handler"
	httpmiddleware "velocity/internal/transport/http/middleware"
	userwsHandler "velocity/internal/transport/userws/handler"
	wsHandler "velocity/internal/transport/ws/handler"
	"velocity/internal/userstream"
	"velocity/pkg/snowflake"
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

	DB   *pgxpool.Pool
	HTTP *fiber.App

	// --------------------------------------------------
	// Utilities
	// --------------------------------------------------

	IDGenerator *snowflake.Generator

	// --------------------------------------------------
	// gRPC Clients
	// --------------------------------------------------

	IdentityClient *identityclient.Client

	// HTTP Middleware
	AuthMiddleware *httpmiddleware.AuthMiddleware

	// --------------------------------------------------
	// Repositories
	// --------------------------------------------------

	UserRepository     repository.UserRepository
	OrderRepository    repository.OrderRepository
	TradeRepository    repository.TradeRepository
	PositionRepository repository.PositionRepository
	SymbolRepository   repository.SymbolRepository
	WalletRepository   repository.WalletRepository
	SellerRepository   repository.SellerRepository
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

	WSHandler     *wsHandler.Handler
	UserWSHandler *userwsHandler.Handler

	// --------------------------------------------------
	// User Stream
	// --------------------------------------------------

	UserHub        *userstream.Hub
	UserPublisher  *userstream.Publisher
	UserDispatcher *userstream.Dispatcher

	// --------------------------------------------------
	// Matching Engine
	// --------------------------------------------------

	Registry         *registry.Registry
	Recovery         *recovery.Recovery
	SnapshotRecovery *recovery.SnapshotRecovery
	WALManager       *wal.Manager

	// --------------------------------------------------
	// Services
	// --------------------------------------------------

	RiskService       *riskservice.Service
	WalletService     *walletservice.Service
	OrderService      *orderservice.Service
	SettlementService *settlementservice.Service
	MarketService     *marketservice.Service
	PositionService   *positionservice.Service
	UserService       *userservice.Service

	VelocityGRPCServer *grpcserver.UserServer
	GRPCServer         *grpcserver.Server

	// --------------------------------------------------
	// HTTP Handlers
	// --------------------------------------------------
	OrderHandler      *handler.OrderHandler
	MarketDataHandler *handler.MarketDataHandler
	WalletHandler     *handler.WalletHandler
	PositionHandler   *handler.PositionHandler
	SellerHandler     *handler.SellerHandler

	MarketStatsManager *stats.Manager
	MarketStatsService *stats.Service
	CandleManager      *candles.Manager
	CandleService      *candles.Service
}
