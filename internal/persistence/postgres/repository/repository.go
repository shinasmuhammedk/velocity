package repository

import (
	"context"
	"velocity/internal/persistence/postgres/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserRepository interface {
	Create(ctx context.Context, params generated.CreateUserParams) (generated.User, error)
	GetByID(ctx context.Context, id int64) (generated.User, error)
	GetByEmail(ctx context.Context, email string) (generated.User, error)
	Exists(ctx context.Context, id int64) (bool, error)
}

type OrderRepository interface {
	Create(ctx context.Context, params generated.CreateOrderParams) (generated.Order, error)
	GetByID(ctx context.Context, id int64) (generated.Order, error)
	UpdateStatus(ctx context.Context, params generated.UpdateOrderStatusParams) error
	ListByUser(ctx context.Context, userID int64) ([]generated.Order, error)
	ListOpenOrders(ctx context.Context, symbol string) ([]generated.Order, error)
	RecoveryOrders(ctx context.Context) ([]generated.Order, error)
	GetPendingStopOrders(ctx context.Context) ([]generated.Order, error)
	UpdateOrderForModify(ctx context.Context, params generated.UpdateOrderForModifyParams) error
	WithTx(tx pgx.Tx) OrderRepository
	UpdateOrderAfterTrade(ctx context.Context, params generated.UpdateOrderAfterTradeParams) error
	ListOpenOrdersByUser(ctx context.Context, userID int64) ([]generated.Order, error)
	ListOrdersByUser(ctx context.Context, userID int64) ([]generated.Order, error)
	GetByUserAndID(ctx context.Context, params generated.GetOrderByUserAndIDParams) (generated.Order, error)
}

type TradeRepository interface {
	Create(ctx context.Context, params generated.CreateTradeParams) (generated.Trade, error)
	CreateIfNotExists(ctx context.Context, params generated.CreateTradeIfNotExistsParams) (generated.Trade, error)
	ListByUser(ctx context.Context, userID int64) ([]generated.Trade, error)
	ListBySymbol(ctx context.Context, symbol string) ([]generated.Trade, error)
	ListBySymbolAsc(ctx context.Context, symbol string) ([]generated.Trade, error)
	GetByID(ctx context.Context, id int64) (generated.Trade, error)
	WithTx(tx pgx.Tx) TradeRepository
	TradeExists(ctx context.Context, id int64) (bool, error)
}

type SymbolRepository interface {
	Create(ctx context.Context, params generated.CreateSymbolParams) (generated.Symbol, error)
	Get(ctx context.Context, symbol string) (generated.Symbol, error)
	List(ctx context.Context) ([]generated.Symbol, error)
	ListActive(ctx context.Context) ([]generated.Symbol, error)
	GetBySymbol(ctx context.Context, symbol string) (generated.Symbol, error)
	UpdateStatus(ctx context.Context, params generated.UpdateSymbolStatusParams) error
}

type PositionRepository interface {
	Upsert(ctx context.Context, params generated.UpsertPositionParams) error
	Get(ctx context.Context, userID int64, symbol string) (generated.Position, error)
	ListByUser(ctx context.Context, userID int64) ([]generated.Position, error)
	WithTx(tx pgx.Tx) PositionRepository
}

type WalletRepository interface {
	Create(ctx context.Context, params generated.CreateWalletParams) (generated.Wallet, error)
	Get(ctx context.Context, userID int64, asset string) (generated.Wallet, error)
	Update(ctx context.Context, params generated.UpdateWalletParams) error
	LockFunds(ctx context.Context, walletID uuid.UUID, amount int64) error
	List(ctx context.Context, userID int64) ([]generated.Wallet, error)
	WithTx(tx pgx.Tx) WalletRepository
}

type FailedSettlementRepository interface {
	Create(ctx context.Context, params generated.CreateFailedSettlementParams) (generated.FailedSettlement, error)
	Get(ctx context.Context, id uuid.UUID) (generated.FailedSettlement, error)
	ListUnresolved(ctx context.Context) ([]generated.FailedSettlement, error)
	IncrementRetryCount(ctx context.Context, id uuid.UUID) error
	Resolve(ctx context.Context, id uuid.UUID) error
}
