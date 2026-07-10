package order

import (
	"time"
	"velocity/pkg/constants"
)

type Order struct {
	// Unique order identifier
	ID string

	// Trader who submitted the order
	UserID string

	// Trading symbol
	// Example: BTCUSDT, AAPL, RELIANCE
	Symbol string

	// BUY or SELL
	Side constants.OrderSide

	// LIMIT or MARKET
	Type constants.OrderType

	// GTC, IOC, FOK
	TimeInForce constants.TimeInForce

	// OPEN, FILLED, CANCELLED etc
	Status constants.OrderStatus

	// Limit price
	// Ignored for MARKET orders
	Price int64

	// Original quantity submitted
	Quantity int64

	// Quantity remaining to be matched
	Remaining int64

	// Total quantity already executed
	Filled int64

	// Timestamp used for price-time priority
	CreatedAt time.Time

	// Updated whenever order state changes
	UpdatedAt time.Time
}
