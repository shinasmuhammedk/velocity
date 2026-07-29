package candles

import "fmt"

func (m *Manager) Get(
	symbol string,
	interval Interval,
) ([]*Candle, bool) {

	m.mu.RLock()
	defer m.mu.RUnlock()

	fmt.Println("symbols:", len(m.candles))
	fmt.Println("looking for:", symbol)

	intervals, ok := m.candles[symbol]
	fmt.Println("symbol exists:", ok)

	if !ok {
		return nil, false
	}

	fmt.Println("intervals:", len(intervals))

	candles, ok := intervals[interval]
	fmt.Println("interval exists:", ok)

	if !ok {
		return nil, false
	}

	fmt.Println("candles:", len(candles))

	out := make([]*Candle, len(candles))
	copy(out, candles)

	return out, true
}