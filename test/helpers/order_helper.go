package testhelpers

import (
	"velocity/internal/domain/order"
	"velocity/pkg/constants"
	"velocity/pkg/timeutil"
)

func NewOrder(
	id string,
	userID string,
	side constants.OrderSide,
	price int64,
	qty int64,
) *order.Order {
	return &order.Order{
		ID:          id,
		UserID:      userID,
		Symbol:      "BTCUSDT",
		Side:        side,
		Type:        constants.OrderTypeLimit,
		TimeInForce: constants.TimeInForceGTC,
		Status:      constants.OrderStatusOpen,
		Price:       price,
		Quantity:    qty,
		Remaining:   qty,
		CreatedAt:   timeutil.UTCNow(),
	}
}