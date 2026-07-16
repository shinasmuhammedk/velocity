package order

import (
	"time"
	"velocity/pkg/constants"
)

type Order struct {
	ID     string
	UserID string
	Symbol string

	Side        constants.OrderSide
	Type        constants.OrderType
	TimeInForce constants.TimeInForce
	Status      constants.OrderStatus

	Price int64

	// Used only by STOP orders
	StopPrice int64

	Quantity  int64
	Remaining int64
	Filled    int64

	CreatedAt time.Time
	UpdatedAt time.Time
}
