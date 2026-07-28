package mapper

import (
	"velocity/internal/domain/order"
	"velocity/internal/persistence/postgres/generated"
	"velocity/pkg/constants"
)

func ToDomainOrder(g generated.Order) *order.Order {
	return &order.Order{
		ID:     g.ID.String(),
		UserID: g.UserID.String(),
		Symbol: g.Symbol,

		Side: constants.OrderSide(g.Side),
		Type: constants.OrderType(g.OrderType),

		TimeInForce: constants.TimeInForce(g.TimeInForce),
		Status:       constants.OrderStatus(g.Status),

		Price:     g.Price.Int64,
		StopPrice: g.StopPrice,

		Quantity:  g.Quantity,
		Remaining: g.Remaining,
		Filled:    g.Filled,

		CreatedAt: g.CreatedAt,
		UpdatedAt: g.UpdatedAt,
	}
}

func ToDomainOrders(rows []generated.Order) []*order.Order {

	orders := make([]*order.Order, 0, len(rows))

	for _, row := range rows {
		orders = append(orders, ToDomainOrder(row))
	}

	return orders
}