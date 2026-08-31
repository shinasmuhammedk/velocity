package request

type WithdrawWalletRequest struct {
	// UserID string `json:"user_id"`
	Asset  string `json:"asset"`
	Amount int64  `json:"amount"`
}
