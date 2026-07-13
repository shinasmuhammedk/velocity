package trade

import (
	"time"

	"github.com/google/uuid"
)

type Trade struct {
    ID uuid.UUID

    BuyOrderID  string
    SellOrderID string

    BuyerID  string
    SellerID string

    Symbol string
    Price int64
    Quantity int64

    ExecutedAt time.Time
}