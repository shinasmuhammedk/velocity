package response

import "velocity/internal/domain/depth"

type OrderBookResponse struct {
	Symbol string        `json:"symbol"`
	Bids   []depth.Level `json:"bids"`
	Asks   []depth.Level `json:"asks"`
}
