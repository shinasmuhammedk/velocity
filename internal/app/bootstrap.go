package app

import (
	"context"

	"velocity/internal/engine/orderbook"
	"velocity/internal/engine/recovery"
	"velocity/internal/engine/registry"
	"velocity/internal/engine/snapshot"
	"velocity/internal/engine/wal"
	"velocity/internal/infrastructure/metrics"
	"velocity/internal/marketdata"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/persistence/postgres/tx"
	"velocity/internal/persistence/worker"
	"velocity/internal/service/orderservice"
	"velocity/internal/transport/http/handler"
	"velocity/internal/transport/http/router"
	wsHandler "velocity/internal/transport/ws/handler"
	wsRouter "velocity/internal/transport/ws/router"
	"velocity/internal/userstream"

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

	// Register repositories
	container.UserRepository = repository.NewUserRepository(container.DB)
	container.OrderRepository = repository.NewOrderRepository(container.DB)
	container.TradeRepository = repository.NewTradeRepository(container.DB)
	container.PositionRepository = repository.NewPositionRepository(container.DB)
	container.SymbolRepository = repository.NewSymbolRepository(container.DB)

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

	container.MarketBroadcaster = marketdata.NewBroadcaster(
		container.MarketPublisher,
	)
	container.Logger.Info("market data broadcaster initialized")

	

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
	)

	container.TradeConsumer = worker.NewTradeConsumer(
		container.TradeWorker,
		container.MarketBroadcaster,
		provider,
	)
	container.Logger.Info("trade consumer initialized")

	// 4. Inject consumer into registry
	container.Registry.SetConsumer(
		container.TradeConsumer,
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

	//OrderService
	container.OrderService = orderservice.New(
		container.OrderRepository,
		container.SymbolRepository,
		container.UserRepository,
		container.Registry,
		container.Logger,
		container.UserDispatcher,
	)
	container.Logger.Info("order service initialized")

	container.WSHandler = wsHandler.NewHandler(container.MarketHub)
	container.Logger.Info("websocket handler initialized")

	//OrderHandler
	container.OrderHandler = handler.NewOrderHandler(
		container.OrderService,
	)
	container.Logger.Info("order handler initialized")

	//router
	router.Register(
		container.HTTP,
		container.OrderHandler,
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

	container.Logger.Info("application bootstrap completed")

	return container, nil
}
