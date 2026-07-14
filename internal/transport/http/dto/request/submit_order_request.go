package request

type SubmitOrderRequest struct {
	UserID string `json:"user_id"`

	Symbol string `json:"symbol"`

	Side string `json:"side"`

	Type string `json:"type"`

	TimeInForce string `json:"time_in_force"`

	Price int64 `json:"price"`

	StopPrice int64 `json:"stop_price"`

	Quantity int64 `json:"quantity"`
}