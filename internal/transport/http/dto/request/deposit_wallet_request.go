package request

type DepositWalletRequest struct {
	Asset  string `json:"asset"`
	Amount int64  `json:"amount"`
}