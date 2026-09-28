package response

import "time"

type WalletTransactionResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Asset     string    `json:"asset"`
	Amount    int64     `json:"amount"`
	Type      string    `json:"type"`
	TradeID   *int64    `json:"trade_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
