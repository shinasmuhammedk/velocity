package wal

import (
	"sync"
)

type Manager struct {
	directory  string
	serializer Serializer

	mu sync.RWMutex

	writers map[string]*Writer
}

func NewManager(
	directory string,
	serializer Serializer,
) *Manager {
	return &Manager{
		directory:  directory,
		serializer: serializer,
		writers:    make(map[string]*Writer),
	}
}

// Writer returns the WAL writer for a symbol.
// It lazily creates one if it doesn't already exist.
func (m *Manager) Writer(
	symbol string,
) (*Writer, error) {

	// Fast path
	m.mu.RLock()
	writer, ok := m.writers[symbol]
	m.mu.RUnlock()

	if ok {
		return writer, nil
	}

	// Slow path
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check
	if writer, ok := m.writers[symbol]; ok {
		return writer, nil
	}

	writer, err := NewWriter(
		m.directory,
		symbol,
		m.serializer,
	)
	if err != nil {
		return nil, err
	}

	m.writers[symbol] = writer

	return writer, nil
}

// Close closes every WAL writer.
func (m *Manager) Close() error {

	m.mu.Lock()
	defer m.mu.Unlock()

	for symbol, writer := range m.writers {

		if err := writer.Close(); err != nil {
			return err
		}

		delete(m.writers, symbol)
	}

	return nil
}