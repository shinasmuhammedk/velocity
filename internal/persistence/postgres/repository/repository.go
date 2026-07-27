package repository

import (
	"context"
	"velocity/internal/persistence/postgres/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserRepository interface {
	Create(ctx context.Context, params generated.CreateUserParams) (generated.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (generated.User, error)
	GetByEmail(ctx context.Context, email string) (generated.User, error)
}

type OrderRepository interface {
	Create(ctx context.Context, params generated.CreateOrderParams) (generated.Order, error)
	GetByID(ctx context.Context, id uuid.UUID) (generated.Order, error)
	UpdateStatus(ctx context.Context, params generated.UpdateOrderStatusParams) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]generated.Order, error)
	ListOpenOrders(ctx context.Context, symbol string) ([]generated.Order, error)
	RecoveryOrders(ctx context.Context) ([]generated.Order, error)
	GetPendingStopOrders(ctx context.Context) ([]generated.Order, error)
	UpdateOrderForModify(ctx context.Context, params generated.UpdateOrderForModifyParams) error
	WithTx(tx pgx.Tx) OrderRepository
	UpdateOrderAfterTrade(ctx context.Context, params generated.UpdateOrderAfterTradeParams) error
}

type TradeRepository interface {
	Create(ctx context.Context, params generated.CreateTradeParams) (generated.Trade, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]generated.Trade, error)
	ListBySymbol(ctx context.Context, symbol string) ([]generated.Trade, error)
	GetByID(ctx context.Context, id uuid.UUID) (generated.Trade, error)
	WithTx(tx pgx.Tx) TradeRepository
	TradeExists(ctx context.Context, id uuid.UUID) (bool, error)
}

type SymbolRepository interface {
	Create(ctx context.Context, params generated.CreateSymbolParams) (generated.Symbol, error)
	Get(ctx context.Context, symbol string) (generated.Symbol, error)
	List(ctx context.Context) ([]generated.Symbol, error)
	ListActive(ctx context.Context) ([]generated.Symbol, error)
	GetBySymbol(ctx context.Context, symbol string) (generated.Symbol, error)
}

type PositionRepository interface {
	Upsert(ctx context.Context, params generated.UpsertPositionParams) error
	Get(ctx context.Context, userID uuid.UUID, symbol string) (generated.Position, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]generated.Position, error)
	WithTx(tx pgx.Tx) PositionRepository
}

type WalletRepository interface {
	Create(ctx context.Context, params generated.CreateWalletParams) (generated.Wallet, error)
	Get(ctx context.Context, userID uuid.UUID, asset string) (generated.Wallet, error)
	Update(ctx context.Context, params generated.UpdateWalletParams) error
	List(ctx context.Context, userID uuid.UUID) ([]generated.Wallet, error)
	WithTx(tx pgx.Tx) WalletRepository
}
