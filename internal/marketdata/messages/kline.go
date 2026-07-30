package messages

import "velocity/internal/analytics/candles"

type KlineMessage struct {
	Event string          `json:"event"`
	Data  *candles.Candle `json:"data"`
}
