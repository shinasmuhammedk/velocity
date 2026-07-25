package settlementservice

import "github.com/google/uuid"

type SettlementRequest struct {
	TradeID uuid.UUID

	BuyOrderID  uuid.UUID
	SellOrderID uuid.UUID

	BuyerID  uuid.UUID
	SellerID uuid.UUID

	Symbol string

	BaseAsset  string
	QuoteAsset string

	Price    int64
	Quantity int64
}
