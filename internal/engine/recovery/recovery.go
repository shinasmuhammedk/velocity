package recovery

import (
	"context"

	"velocity/internal/domain/order"
	"velocity/internal/engine/registry"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
	"velocity/pkg/constants"

	"go.uber.org/zap"
)

type Recovery struct {
	orderRepo repository.OrderRepository
	registry  *registry.Registry
	logger    *zap.Logger
}

func New(
	orderRepo repository.OrderRepository,
	registry *registry.Registry,
	logger *zap.Logger,
) *Recovery {
	return &Recovery{
		orderRepo: orderRepo,
		registry:  registry,
		logger:    logger,
	}
}

func (r *Recovery) Load(
	ctx context.Context,
) error {

	orders, err := r.orderRepo.RecoveryOrders(ctx)
	if err != nil {
		return err
	}

	for _, dbOrder := range orders {

		engine := r.registry.Get(
			dbOrder.Symbol,
		)

		engine.RecoverOrder(
			toDomainOrder(dbOrder),
		)
	}

	r.logger.Info(
		"recovery completed",
		zap.Int(
			"orders_recovered",
			len(orders),
		),
	)

	return nil
}

func toDomainOrder(
	dbOrder generated.Order,
) *order.Order {

	price := int64(0)
	if dbOrder.Price.Valid {
		price = dbOrder.Price.Int64
	}

	return &order.Order{
		ID:          dbOrder.ID.String(),
		UserID:      dbOrder.UserID.String(),
		Symbol:      dbOrder.Symbol,
		Side:        constants.OrderSide(dbOrder.Side),
		Type:        constants.OrderType(dbOrder.OrderType),
		TimeInForce: constants.TimeInForce(dbOrder.TimeInForce),
		Status:      constants.OrderStatus(dbOrder.Status),
		Price:       price,
		StopPrice:   dbOrder.StopPrice,
		Quantity:    dbOrder.Quantity,
		Remaining:   dbOrder.Remaining,
		Filled:      dbOrder.Filled,
		CreatedAt:   dbOrder.CreatedAt,
		UpdatedAt:   dbOrder.UpdatedAt,
	}
}