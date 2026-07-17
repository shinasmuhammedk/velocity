package snapshot

import (
    "time"
    "velocity/internal/domain/order"
)

type Snapshot struct {
    Symbol         string          `json:"symbol"`
    Sequence        uint64          `json:"sequence"`
    LastTradePrice  int64           `json:"last_trade_price"`
    CreatedAt       time.Time       `json:"created_at"`

    ActiveOrders []*order.Order `json:"active_orders"`
    StopOrders   []*order.Order `json:"stop_orders"`
}