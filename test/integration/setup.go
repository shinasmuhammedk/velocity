package integration

import (
	"context"
	"testing"

	"velocity/internal/config"
	"velocity/internal/persistence/postgres"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/persistence/postgres/tx"

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
}

func NewTestContext(t *testing.T) *TestContext {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err)

	db, err := postgres.New(cfg.Database)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	return &TestContext{
		Ctx: context.Background(),
		DB:  db,
        
        TxManager: tx.NewManager(db),

		UserRepo:     repository.NewUserRepository(db),
		OrderRepo:    repository.NewOrderRepository(db),
		TradeRepo:    repository.NewTradeRepository(db),
		WalletRepo:   repository.NewWalletRepository(db),
		PositionRepo: repository.NewPositionRepository(db),
		SymbolRepo:   repository.NewSymbolRepository(db),
	}
}