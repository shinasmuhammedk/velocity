package testhelpers

import (
	"time"

	"velocity/internal/domain/order"
	"velocity/pkg/constants"
)

// NewOrder builds a simple open, GTC limit order for use in tests.
func NewOrder(id int64, userID int64, side constants.OrderSide, price int64, quantity int64) *order.Order {
	return &order.Order{
		ID:          id,
		UserID:      userID,
		Symbol:      "BTCUSDT",
		Side:        side,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       price,
		Quantity:    quantity,
		Remaining:   quantity,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}
}
