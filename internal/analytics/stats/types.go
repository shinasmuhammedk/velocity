package stats

import "time"

type MarketStats struct {
	Symbol string `json:"symbol"`

	OpenPrice int64 `json:"open_price"`
	HighPrice int64 `json:"high_price"`
	LowPrice  int64 `json:"low_price"`
	LastPrice int64 `json:"last_price"`

	BaseVolume  int64 `json:"base_volume"`
	QuoteVolume int64 `json:"quote_volume"`

	TradeCount uint64 `json:"trade_count"`

	// Internal field for managing the 24-hour statistics window.
	// Omit it from JSON responses.
	WindowStart time.Time `json:"-"`
}
