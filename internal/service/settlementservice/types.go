package settlementservice

type SettlementRequest struct {
	TradeID int64

	BuyOrderID  int64
	SellOrderID int64

	BuyerID  int64
	SellerID int64

	Symbol string

	BaseAsset  string
	QuoteAsset string

	Price    int64
	Quantity int64
}
