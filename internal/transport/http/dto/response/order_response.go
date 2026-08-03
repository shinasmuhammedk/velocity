package response

import "time"

type OrderResponse struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Symbol       string    `json:"symbol"`
	Side         string    `json:"side"`
	Type         string    `json:"type"`
	TimeInForce  string    `json:"time_in_force"`
	Status       string    `json:"status"`
	Price        int64     `json:"price"`
	StopPrice    int64     `json:"stop_price"`
	Quantity     int64     `json:"quantity"`
	Remaining    int64     `json:"remaining"`
	Filled       int64     `json:"filled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}