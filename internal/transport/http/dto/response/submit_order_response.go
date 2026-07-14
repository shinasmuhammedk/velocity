package response

type SubmitOrderResponse struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
	Symbol  string `json:"symbol"`
}