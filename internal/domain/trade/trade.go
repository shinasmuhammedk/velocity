package trade

import (
	"time"

)

type Trade struct {
    ID int64

    BuyOrderID  int64
    SellOrderID int64

    BuyerID  int64
    SellerID int64

    Symbol string
    Price int64
    Quantity int64

    ExecutedAt time.Time
}