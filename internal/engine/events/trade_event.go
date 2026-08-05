package events

type TradeExecutedEvent struct {
	BaseEvent

	TradeID int64

	BuyOrderID int64
	SellOrderID int64

	BuyerID int64
	SellerID int64

	Symbol string

	Price int64

	Quantity int64
}

func (e TradeExecutedEvent) Type() EventType {
	return TradeExecutedEventType
}