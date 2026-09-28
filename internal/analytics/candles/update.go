package candles

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

	intervals, ok := m.candles[symbol]
	if !ok {
		intervals = make(map[Interval][]*Candle)
		m.candles[symbol] = intervals
	}

	for _, interval := range SupportedIntervals {
		m.updateInterval(
			intervals,
			symbol,
			interval,
			price,
			quantity,
			now,
		)
	}
}

func (m *Manager) updateInterval(
	intervals map[Interval][]*Candle,
	symbol string,
	interval Interval,
	price int64,
	quantity int64,
	now time.Time,
) {
	candles := intervals[interval]

	current := last(candles)

	start := now.Truncate(interval.Duration())
	end := start.Add(interval.Duration())

	if current == nil || !current.OpenTime.Equal(start) {
		// The previous candle is now closed because the new trade
		// belongs to a different interval.
		if current != nil {
			closed := *current

			select {
			case m.closed <- &closed:
			default:
				// Never block the candle update path.
				// Persistence must not slow down trade processing.
			}
		}

		current = &Candle{
			Symbol: symbol,

			Interval: interval,

			OpenTime:  start,
			CloseTime: end,

			Open:  price,
			High:  price,
			Low:   price,
			Close: price,

			Volume:      quantity,
			QuoteVolume: price * quantity,

			TradeCount: 1,
		}

		intervals[interval] = append(
			candles,
			current,
		)

		return
	}

	current.Close = price

	if price > current.High {
		current.High = price
	}

	if price < current.Low {
		current.Low = price
	}

	current.Volume += quantity
	current.QuoteVolume += price * quantity
	current.TradeCount++
}
