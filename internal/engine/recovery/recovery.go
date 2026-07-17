package recovery

import (
	"context"

	"velocity/internal/domain/order"
	"velocity/internal/engine/registry"
	"velocity/internal/persistence/postgres/generated"
	"velocity/pkg/constants"

	"go.uber.org/zap"
)

type Recovery struct {
	orderRepo RecoveryOrderRepository
	registry  *registry.Registry
	logger    *zap.Logger
}

type RecoveryOrderRepository interface {
	RecoveryOrders(
		ctx context.Context,
	) ([]generated.Order, error)
}



func New(
	orderRepo RecoveryOrderRepository,
	registry *registry.Registry,
	logger *zap.Logger,
) *Recovery {
	return &Recovery{
		orderRepo: orderRepo,
		registry:  registry,
		logger:    logger,
	}
}

// Load replays open orders from Postgres into their engines.
// alreadyRestored marks symbols whose order book was already fully
// rebuilt from a snapshot this startup — those symbols are skipped here
// entirely, since replaying them too would insert every open order a
// second time (see recovery double-insertion bug).
func (r *Recovery) Load(
	ctx context.Context,
	alreadyRestored map[string]bool,
) error {

	orders, err := r.orderRepo.RecoveryOrders(ctx)
	if err != nil {
		return err
	}

	recoveredCount := 0
	skippedCount := 0

	for _, dbOrder := range orders {

		if alreadyRestored[dbOrder.Symbol] {
			skippedCount++
			continue
		}

		engine := r.registry.Get(dbOrder.Symbol)

		engine.RecoverOrder(
			toDomainOrder(dbOrder),
		)

		recoveredCount++
	}

	r.logger.Info(
		"recovery completed",
		zap.Int("orders_recovered", recoveredCount),
		zap.Int("orders_skipped_snapshot_already_restored", skippedCount),
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