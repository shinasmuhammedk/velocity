package stats

import (
	"sync"
)

type Manager struct {
	mu    sync.RWMutex
	stats map[string]*MarketStats
}

func NewManager() *Manager {
	return &Manager{
		stats: make(map[string]*MarketStats),
	}
}

func (m *Manager) Get(symbol string) (*MarketStats, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
    

	s, ok := m.stats[symbol]
	return s, ok
}

func (m *Manager) Set(symbol string, stats *MarketStats) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats[symbol] = stats
}

func (m *Manager) Snapshot() map[string]*MarketStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make(map[string]*MarketStats, len(m.stats))

	for k, v := range m.stats {
		copy := *v
		out[k] = &copy
	}

	return out
}