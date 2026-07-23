package events

type TradeExecutedEvent struct {
	BaseEvent

	TradeID string

	BuyOrderID string
	SellOrderID string

	BuyerID string
	SellerID string

	Symbol string

	Price int64

	Quantity int64
}

func (e TradeExecutedEvent) Type() EventType {
	return TradeExecutedEventType
}