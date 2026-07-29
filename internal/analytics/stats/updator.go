package stats

import (
	"time"
)

func (m *Manager) Update(
	symbol string,
	price int64,
	quantity int64,
	now time.Time,
) {

	m.mu.Lock()
	defer m.mu.Unlock()

	stats, ok := m.stats[symbol]

	if !ok {

		stats = &MarketStats{
			Symbol: symbol,

			OpenPrice: price,
			HighPrice: price,
			LowPrice:  price,
			LastPrice: price,

			BaseVolume:  quantity,
			QuoteVolume: price * quantity,

			TradeCount: 1,

			WindowStart: now,
		}

		m.stats[symbol] = stats

		return
	}

	if stats.WindowExpired(now) {

		stats.OpenPrice = price
		stats.HighPrice = price
		stats.LowPrice = price
		stats.LastPrice = price

		stats.BaseVolume = quantity
		stats.QuoteVolume = price * quantity

		stats.TradeCount = 1
		stats.WindowStart = now

		return
	}

	stats.LastPrice = price

	if price > stats.HighPrice {
		stats.HighPrice = price
	}

	if price < stats.LowPrice {
		stats.LowPrice = price
	}

	stats.BaseVolume += quantity
	stats.QuoteVolume += price * quantity
	stats.TradeCount++
}
