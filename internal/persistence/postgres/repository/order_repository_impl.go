package repository

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

	"github.com/jackc/pgx/v5"
)

type orderRepository struct {
	queries *generated.Queries
}

func NewOrderRepository(db generated.DBTX) OrderRepository {
	return &orderRepository{
		queries: generated.New(db),
	}
}


func (r *orderRepository) WithTx(tx pgx .Tx) OrderRepository {
	return &orderRepository{
		queries: generated.New(tx),
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
	id int64,
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
	userID int64,
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

func (r *orderRepository) UpdateOrderForModify(
    ctx context.Context,
    params generated.UpdateOrderForModifyParams,
) error {
    return r.queries.UpdateOrderForModify(
        ctx,
        params,
    )
}

func (r *orderRepository) UpdateOrderAfterTrade(
    ctx context.Context,
    params generated.UpdateOrderAfterTradeParams,
) error {
    return r.queries.UpdateOrderAfterTrade(
        ctx,
        params,
    )
}

func (r *orderRepository) ListOpenOrdersByUser(
	ctx context.Context,
	userID int64,
) ([]generated.Order, error) {

	return r.queries.ListOpenOrdersByUser(
		ctx,
		userID,
	)
}


func (r *orderRepository) ListOrdersByUser(
    ctx context.Context,
    userID int64,
) ([]generated.Order, error) {

    return r.queries.ListOrdersByUser(
        ctx,
        userID,
    )
}

func (r *orderRepository) GetByUserAndID(
	ctx context.Context,
	params generated.GetOrderByUserAndIDParams,
) (generated.Order, error) {

	return r.queries.GetOrderByUserAndID(
		ctx,
		params,
	)
}