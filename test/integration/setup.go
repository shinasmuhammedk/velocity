package integration

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"velocity/internal/config"
	"velocity/internal/persistence/postgres"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/persistence/postgres/tx"
	"velocity/internal/userstream"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type TestContext struct {
	Ctx context.Context
	DB  *pgxpool.Pool

	TxManager tx.Manager

	UserRepo     repository.UserRepository
	OrderRepo    repository.OrderRepository
	TradeRepo    repository.TradeRepository
	WalletRepo   repository.WalletRepository
	PositionRepo repository.PositionRepository
	SymbolRepo   repository.SymbolRepository

	UserDispatcher *userstream.Dispatcher
}

func NewTestContext(t *testing.T) *TestContext {
	t.Helper()

	_, currentFile, _, _ := runtime.Caller(0)

	projectRoot := filepath.Join(
		filepath.Dir(currentFile),
		"..",
		"..",
	)

	configPath := filepath.Join(
		projectRoot,
		"configs",
		"config.development.yaml",
	)

	cfg, err := config.LoadFromPath(configPath)
	require.NoError(t, err)

	db, err := postgres.New(cfg.Database)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	hub := userstream.NewHub()
	publisher := userstream.NewPublisher(hub)
	dispatcher := userstream.NewDispatcher(publisher)

	return &TestContext{
		Ctx: context.Background(),
		DB:  db,

		TxManager: tx.NewManager(db),

		UserRepo:       repository.NewUserRepository(db),
		OrderRepo:      repository.NewOrderRepository(db),
		TradeRepo:      repository.NewTradeRepository(db),
		WalletRepo:     repository.NewWalletRepository(db),
		PositionRepo:   repository.NewPositionRepository(db),
		SymbolRepo:     repository.NewSymbolRepository(db),
		UserDispatcher: dispatcher,
	}
}
