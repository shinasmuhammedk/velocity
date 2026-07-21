package userstream

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type ExecutionReport struct {
	OrderID string `json:"order_id"`

	Symbol string `json:"symbol"`

	Status string `json:"status"`

	Price int64 `json:"price"`

	Quantity int64 `json:"quantity"`

	FilledQuantity int64 `json:"filled_quantity"`

	RemainingQuantity int64 `json:"remaining_quantity"`
}

type BalanceUpdate struct {
	Asset string `json:"asset"`

	Available int64 `json:"available"`

	Locked int64 `json:"locked"`
}

type PositionUpdate struct {
	Symbol string `json:"symbol"`

	Quantity int64 `json:"quantity"`

	AvgPrice int64 `json:"avg_price"`

	PnL int64 `json:"pnl"`
}