package repository

import (
	"context"
	"velocity/internal/persistence/postgres/generated"

	"github.com/google/uuid"
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
}

type TradeRepository interface {
	Create(ctx context.Context, params generated.CreateTradeParams) (generated.Trade, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]generated.Trade, error)
	ListBySymbol(ctx context.Context, symbol string) ([]generated.Trade, error)
	GetByID(ctx context.Context, id uuid.UUID) (generated.Trade, error)
}

type SymbolRepository interface {
	Create(ctx context.Context, params generated.CreateSymbolParams) (generated.Symbol, error)
	Get(ctx context.Context, symbol string) (generated.Symbol, error)
	List(ctx context.Context) ([]generated.Symbol, error)
	ListActive(ctx context.Context) ([]generated.Symbol, error)
}

type PositionRepository interface {
	Upsert(ctx context.Context, params generated.UpsertPositionParams) error
	Get(ctx context.Context, userID uuid.UUID, symbol string) (generated.Position, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]generated.Position, error)
}
