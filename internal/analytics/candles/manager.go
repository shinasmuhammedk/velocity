package candles

import (
	"fmt"
	"sync"
)

var SupportedIntervals = []Interval{
	Interval1m,
	Interval5m,
	Interval15m,
	Interval1h,
	Interval4h,
	Interval1d,
}

type Manager struct {
	mu sync.RWMutex

	candles map[string]map[Interval][]*Candle
}

func NewManager() *Manager {
	return &Manager{
		candles: make(map[string]map[Interval][]*Candle),
	}
}

func last(candles []*Candle) *Candle {
	if len(candles) == 0 {
		return nil
	}

	return candles[len(candles)-1]
}

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



func (m *Manager) Latest(
	symbol string,
	interval Interval,
) (*Candle, bool) {

	m.mu.RLock()
	defer m.mu.RUnlock()

	intervals, ok := m.candles[symbol]
	if !ok {
		return nil, false
	}

	candles, ok := intervals[interval]
	if !ok || len(candles) == 0 {
		return nil, false
	}

	return candles[len(candles)-1], true
}