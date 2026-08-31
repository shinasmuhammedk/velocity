package response

type SubmitOrderResponse struct {
	OrderID int64  `json:"order_id"`
	Status  string `json:"status"`
	Symbol  string `json:"symbol"`
}
