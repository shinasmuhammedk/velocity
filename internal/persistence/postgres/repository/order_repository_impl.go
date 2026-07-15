package repository

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

	"github.com/google/uuid"
)

type orderRepository struct {
	queries *generated.Queries
}

func NewOrderRepository(db generated.DBTX) OrderRepository {
	return &orderRepository{
		queries: generated.New(db),
	}
}

func (r *orderRepository) Create(
	ctx context.Context,
	params generated.CreateOrderParams,
) (generated.Order, error) {
	return r.queries.CreateOrder(ctx, params)
}

func (r *orderRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (generated.Order, error) {
	return r.queries.GetOrderByID(ctx, id)
}

func (r *orderRepository) UpdateStatus(
	ctx context.Context,
	params generated.UpdateOrderStatusParams,
) error {
	return r.queries.UpdateOrderStatus(ctx, params)
}

func (r *orderRepository) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
) ([]generated.Order, error) {
	return r.queries.ListOrdersByUser(ctx, userID)
}

func (r *orderRepository) ListOpenOrders(
	ctx context.Context,
	symbol string,
) ([]generated.Order, error) {
	return r.queries.ListOpenOrders(ctx, symbol)
}

func (r *orderRepository) RecoveryOrders(
    ctx context.Context,
) ([]generated.Order, error) {
    return r.queries.RecoveryOrders(ctx)
}

func (r *orderRepository) GetPendingStopOrders(
    ctx context.Context,
) ([]generated.Order, error) {
    return r.queries.GetPendingStopOrders(ctx)
}