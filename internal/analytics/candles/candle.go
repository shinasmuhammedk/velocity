package candles

import "time"

type Candle struct {
	Symbol string `json:"symbol"`

	Interval Interval `json:"interval"`

	OpenTime  time.Time `json:"open_time"`
	CloseTime time.Time `json:"close_time"`

	Open  int64 `json:"open"`
	High  int64 `json:"high"`
	Low   int64 `json:"low"`
	Close int64 `json:"close"`

	Volume      int64 `json:"volume"`
	QuoteVolume int64 `json:"quote_volume"`

	TradeCount uint64 `json:"trade_count"`
}
