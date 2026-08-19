package app

import (
	"context"
	"velocity/internal/analytics/candles"
	"velocity/internal/analytics/stats"
	"velocity/internal/engine/events"
	"velocity/internal/engine/orderbook"
	"velocity/internal/engine/recovery"
	"velocity/internal/engine/registry"
	"velocity/internal/engine/snapshot"
	"velocity/internal/engine/wal"
	"velocity/internal/infrastructure/kafka"
	"velocity/internal/infrastructure/metrics"
	"velocity/internal/infrastructure/redis"
	"velocity/internal/marketdata"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/persistence/postgres/tx"
	"velocity/internal/persistence/worker"
	"velocity/internal/service/marketservice"
	"velocity/internal/service/orderservice"
	"velocity/internal/service/positionservice"
	"velocity/internal/service/riskservice"
	"velocity/internal/service/settlementservice" // <-- Add this
	"velocity/internal/service/userservice"
	"velocity/internal/service/walletservice"
	"velocity/internal/transport/http/handler"
	"velocity/internal/transport/http/router"
	userwsHandler "velocity/internal/transport/userws/handler"
	userwsRouter "velocity/internal/transport/userws/router"
	wsHandler "velocity/internal/transport/ws/handler"
	wsRouter "velocity/internal/transport/ws/router"
	"velocity/internal/userstream"
	"velocity/pkg/constants"
	"velocity/pkg/snowflake"

	identityclient "velocity/internal/transport/grpc/client/identity"
	httpmiddleware "velocity/internal/transport/http/middleware"

	grpcserver "velocity/internal/transport/grpc/server"

	"github.com/gofiber/adaptor/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Bootstrap creates and initializes the application.
//
// It serves as the composition root of Velocity.
// All application dependencies are wired together here.
func Bootstrap() (*Container, error) {

	container, err := Startup()
	if err != nil {
		return nil, err
	}

	container.Redis = redis.New(container.Config.Redis)
	if err := container.Redis.Ping(context.Background()); err != nil {
		return nil, err
	}
	container.Logger.Info("redis initialized")

	container.KafkaProducer = kafka.NewProducer(
		container.Config.Kafka.Brokers,
		container.Config.Kafka.Topic,
	)
	container.Logger.Info("kafka producer initialized")

	container.MarketCache = redis.NewMarketCache(container.Redis)
	container.Logger.Info("market data cache initialized")

	container.RedisHealth = redis.NewHealthChecker(container.Redis)

	container.IDGenerator = snowflake.New(1)
	container.Logger.Info("snowflake id generator initialized")

	identityClient, err := identityclient.New("localhost:50051")
	if err != nil {
		return nil, err
	}

	container.IdentityClient = identityClient

	container.AuthMiddleware = httpmiddleware.NewAuthMiddleware(identityClient)

	container.Logger.Info("identity grpc client initialized")

	// Register repositories
	container.UserRepository = repository.NewUserRepository(container.DB)
	container.OrderRepository = repository.NewOrderRepository(container.DB)
	container.TradeRepository = repository.NewTradeRepository(container.DB)
	container.PositionRepository = repository.NewPositionRepository(container.DB)
	container.SymbolRepository = repository.NewSymbolRepository(container.DB)
	container.WalletRepository = repository.NewWalletRepository(container.DB)
	container.FailedSettlementRepository = repository.NewFailedSettlementRepository(container.DB)

	container.Logger.Info("repositories initialized")

	//Transaction Manager
	container.TxManager = tx.NewManager(container.DB)
	container.Logger.Info("transaction manager initialized")
	// --------------------------------------------------
	// Future Wiring
	// --------------------------------------------------

	// --------------------------------------------------
	// Market Data
	// --------------------------------------------------

	container.MarketHub = marketdata.NewHub()
	container.Logger.Info("market data hub initialized")

	container.MarketPublisher = marketdata.NewPublisher(
		container.MarketHub,
	)
	container.Logger.Info("market data publisher initialized")

	container.UserHub = userstream.NewHub()

	container.UserPublisher = userstream.NewPublisher(
		container.UserHub,
	)

	container.UserDispatcher = userstream.NewDispatcher(
		container.UserPublisher,
	)

	//workers
	container.TradeWorker = worker.NewTradePersistenceWorker(
		container.TxManager,
		container.OrderRepository,
		container.TradeRepository,
		container.PositionRepository,
	)
	container.Logger.Info("trade persistence worker initialized")

	// Register services
	//
	// Register HTTP handlers
	//

	//metrics
	metrics.Register()
	container.Logger.Info(
		"prometheus metrics registered",
	)

	// Register WebSocket hub
	//

	serializer := snapshot.NewJSONSerializer()

	writer := snapshot.NewWriter(
		"./snapshots",
		serializer,
	)

	walSerializer := wal.NewJSONSerializer()

	container.WALManager = wal.NewManager(
		"./wal",
		walSerializer,
	)

	container.Logger.Info("WAL Manager initialized")

	// Register background workers
	//
	// Matching Engine Registry
	container.Registry = registry.New(
		writer,
		container.WALManager,
	)

	kafkaPublisher := kafka.NewEventPublisher(
		container.KafkaProducer,
		container.Config.Kafka.Topic,
        container.Logger,
	)
    
    container.KafkaEventPublisher = kafkaPublisher

	container.Registry.Publisher().Subscribe(
		events.TradeExecutedEventType,
		kafkaPublisher,
	)

	container.Registry.Publisher().Subscribe(
		events.OrderAcceptedEventType,
		kafkaPublisher,
	)

	container.Registry.Publisher().Subscribe(
		events.OrderRejectedEventType,
		kafkaPublisher,
	)

	container.Registry.Publisher().Subscribe(
		events.OrderCancelledEventType,
		kafkaPublisher,
	)

	container.Registry.Publisher().Subscribe(
		events.OrderModifiedEventType,
		kafkaPublisher,
	)

	container.Registry.Publisher().Subscribe(
		events.OrderTriggeredEventType,
		kafkaPublisher,
	)

	container.Logger.Info("Kafka event publisher registered")

	container.MarketStatsManager = stats.NewManager()
	container.MarketStatsService = stats.NewService(
		container.MarketStatsManager,
	)

	statsSubscriber := stats.NewSubscriber(
		container.MarketStatsManager,
	)

	container.Registry.Publisher().Subscribe(
		events.TradeExecutedEventType,
		statsSubscriber,
	)

	container.Logger.Info("Stats subscriber registered")

	container.CandleManager = candles.NewManager()

	container.CandleService = candles.NewService(
		container.CandleManager,
	)

	container.MarketBroadcaster = marketdata.NewBroadcaster(
		container.MarketPublisher,
		container.CandleService,
	)
	container.Logger.Info("market data broadcaster initialized")

	candleSubscriber := candles.NewSubscriber(
		container.CandleManager,
	)

	container.Registry.Publisher().Subscribe(
		events.TradeExecutedEventType,
		candleSubscriber,
	)
	container.Logger.Info("Candle subscriber registered")

	provider := func(symbol string) *orderbook.OrderBook {
		engine := container.Registry.Get(symbol)

		if engine == nil {
			return nil
		}

		return engine.OrderBook()
	}
	container.Logger.Info("engine registry initialized")

	snapshotLoader := snapshot.NewLoader(
		"./snapshots",
		snapshot.NewJSONSerializer(),
	)

	snapshotRecovery := recovery.NewSnapshotRecovery(
		snapshotLoader,
		container.Registry,
		container.WALManager,
	)

	container.Recovery = recovery.New(
		container.OrderRepository,
		container.Registry,
		container.Logger,
	)
	container.Logger.Info("recovery service initialized")

	symbols, err := container.SymbolRepository.List(
		context.Background(),
	)

	if err != nil {
		return nil, err
	}

	// Snapshot restore runs FIRST, per symbol. Any symbol successfully
	// restored from a snapshot is recorded here, so the DB-based recovery
	// pass below skips it entirely — otherwise every open order for that
	// symbol would be inserted a second time (see recovery bug notes).
	alreadyRestored := make(map[string]bool, len(symbols))

	for _, symbol := range symbols {

		restored, err := snapshotRecovery.Restore(symbol.Symbol)
		if err != nil {
			return nil, err
		}

		alreadyRestored[symbol.Symbol] = restored
	}

	container.Logger.Info("snapshot recovery completed")

	if err := container.Recovery.Load(context.Background(), alreadyRestored); err != nil {
		return nil, err
	}

	container.Logger.Info("database recovery completed")

	container.WalletService = walletservice.New(
		container.WalletRepository,
	)

	container.UserService = userservice.New(
		container.UserRepository,
		container.WalletService,
	)

	container.Logger.Info("user service initialized")

	grpcServer, err := grpcserver.New(container.UserService)
	if err != nil {
		return nil, err
	}

	container.GRPCServer = grpcServer

	container.VelocityGRPCServer = grpcserver.NewUserServer(
		container.UserService,
	)

	container.Logger.Info("velocity grpc server initialized")

	//risk service
	container.RiskService = riskservice.New(
		riskservice.NewQuantityValidator(),
		riskservice.NewPriceValidator(),
		riskservice.NewBalanceValidator(
			container.WalletService,
			container.SymbolRepository,
		),
	)

	container.SettlementService = settlementservice.New(
		container.TxManager,
		container.UserDispatcher,
	)

	container.MarketService = marketservice.New(
		container.Registry,
		container.SymbolRepository,
		container.TradeRepository,
		container.MarketStatsService,
		container.CandleService,
		container.MarketCache,
	)

	container.PositionService = positionservice.New(
		container.PositionRepository,
	)

	container.TradeConsumer = worker.NewTradeConsumer(
		container.SettlementService,
		container.SymbolRepository,
        container.FailedSettlementRepository,
		container.MarketBroadcaster,
		provider,
		container.Logger,
	)
	container.Logger.Info("trade consumer initialized")

	// 4. Inject consumer into registry
	container.Registry.SetConsumer(
		container.TradeConsumer,
	)

	//OrderService
	container.OrderService = orderservice.New(
		container.OrderRepository,
		container.SymbolRepository,
		container.UserRepository,
		container.RiskService,
		container.WalletService,
		container.Registry,
		container.Logger,
		container.UserDispatcher,
		// container.IdentityClient,
		container.IDGenerator,
	)
	container.Logger.Info("order service initialized")

	container.WSHandler = wsHandler.NewHandler(container.MarketHub)
	container.Logger.Info("websocket handler initialized")

	container.UserWSHandler = userwsHandler.NewHandler(container.UserHub)
	container.Logger.Info("private websocket handler initialized")

	//OrderHandler
	container.OrderHandler = handler.NewOrderHandler(
		container.OrderService,
	)
	container.Logger.Info("order handler initialized")

	// MarketDataHandler
	container.MarketDataHandler = handler.NewMarketDataHandler(
		container.MarketService,
	)
	container.Logger.Info("market data handler initialized")

	container.WalletHandler = handler.NewWalletHandler(
		container.WalletService,
	)
	container.PositionHandler = handler.NewPositionHandler(
		container.PositionService,
	)
	container.HealthHandler = handler.NewHealthHandler(
		container.DB,
		container.RedisHealth,
	)
	container.AdminHandler = handler.NewAdminHandler(
		container.MarketService,
	)

	// router
	router.Register(
		container.HTTP,
		// container.IdentityClient,
		container.OrderHandler,
		container.MarketDataHandler,
		container.WalletHandler,
		container.PositionHandler,
		container.HealthHandler,
		container.AdminHandler,
		container.AuthMiddleware.Authenticate,
		httpmiddleware.RequireRole(constants.RoleAdmin),
	)

	container.HTTP.Get(
		"/metrics",
		adaptor.HTTPHandler(promhttp.Handler()),
	)

	// WebSocket Routes
	wsRouter.Register(
		container.HTTP,
		container.WSHandler,
	)

	userwsRouter.Register(
		container.HTTP,
		container.UserWSHandler,
		container.AuthMiddleware,
	)

	container.Logger.Info("application bootstrap completed")

	return container, nil
}
