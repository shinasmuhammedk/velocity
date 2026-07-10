package trade

import "time"

type Trade struct {
	ID string

	BuyOrderID  string
	SellOrderID string

	BuyerID  string
	SellerID string

	Symbol string

	Price int64

	Quantity int64

	ExecutedAt time.Time
}