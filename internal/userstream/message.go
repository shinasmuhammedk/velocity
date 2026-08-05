package userstream

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type ExecutionReport struct {
	OrderID int64 `json:"order_id"`

	Symbol string `json:"symbol"`

	Status string `json:"status"`

	Price int64 `json:"price"`

	Quantity int64 `json:"quantity"`

	FilledQuantity int64 `json:"filled_quantity"`

	RemainingQuantity int64 `json:"remaining_quantity"`
}

type TradeExecution struct {
	TradeID int64 `json:"trade_id"`

	BuyOrderID  int64 `json:"buy_order_id"`
	SellOrderID int64 `json:"sell_order_id"`

	Symbol string `json:"symbol"`

	Price int64 `json:"price"`

	Quantity int64 `json:"quantity"`

	BuyerID  int64 `json:"buyer_id"`
	SellerID int64 `json:"seller_id"`
}

type BalanceUpdate struct {
	Asset string `json:"asset"`

	Available int64 `json:"available"`

	Locked int64 `json:"locked"`
}

type PositionUpdate struct {
	Symbol string `json:"symbol"`

	Quantity int64 `json:"quantity"`
}
