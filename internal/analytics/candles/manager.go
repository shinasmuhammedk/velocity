package candles

import (
	"sync"
	"time"
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

	intervals, ok := m.candles[symbol]
	if !ok {
		return nil, false
	}

	candles, ok := intervals[interval]
	if !ok {
		return nil, false
	}

	out := make([]*Candle, len(candles))
	copy(out, candles)

	return out, true
}

func (m *Manager) Query(
	symbol string,
	interval Interval,
	limit int,
	startTime *time.Time,
	endTime *time.Time,
) ([]*Candle, bool) {

	m.mu.RLock()
	defer m.mu.RUnlock()

	intervals, ok := m.candles[symbol]
	if !ok {
		return nil, false
	}

	candles, ok := intervals[interval]
	if !ok {
		return nil, false
	}

	// We iterate from newest to oldest so that limit means
	// "latest N candles".
	result := make([]*Candle, 0, limit)

	for i := len(candles) - 1; i >= 0; i-- {
		candle := candles[i]

		if startTime != nil && candle.OpenTime.Before(*startTime) {
			break
		}

		if endTime != nil && candle.OpenTime.After(*endTime) {
			continue
		}

		result = append(result, candle)

		if len(result) >= limit {
			break
		}
	}

	// Reverse so the API remains chronological:
	// oldest → newest.
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, true
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
