package app

import (
	"velocity/internal/engine/registry"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/persistence/postgres/tx"
	"velocity/internal/persistence/worker"
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

	//workers
	container.TradeWorker = worker.NewTradePersistenceWorker(
		container.TxManager,
		container.OrderRepository,
		container.TradeRepository,
		container.PositionRepository,
	)

	container.Logger.Info("trade persistence worker initialized")

	container.TradeConsumer = worker.NewTradeConsumer(
		container.TradeWorker,
	)

	container.Logger.Info("trade consumer initialized")
	// Register services
	//
	// Register HTTP handlers
	//
	// Register WebSocket hub
	//
	// Register background workers
	//
	// Matching Engine Registry
	container.Registry = registry.New(
        container.TradeConsumer,
    )

	container.Logger.Info("engine registry initialized")

	container.Logger.Info("application bootstrap completed")

	return container, nil
}
