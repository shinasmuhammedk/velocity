package worker

import "velocity/internal/engine/orderbook"

type OrderBookProvider func(
	symbol string,
) *orderbook.OrderBook
