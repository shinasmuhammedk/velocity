package candles

import "sync"

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