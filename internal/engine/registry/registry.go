package registry

import (
	"context"
	"sync"

	"velocity/internal/engine"
	"velocity/internal/persistence/worker"
)

type Registry struct {
	engines map[string]*engine.Engine
	mu      sync.RWMutex

	consumer *worker.TradeConsumer
}

func New(consumer *worker.TradeConsumer) *Registry {
	return &Registry{
		engines:  make(map[string]*engine.Engine),
		consumer: consumer,
	}
}

// Get returns the engine for a symbol.
// If it does not exist, it creates one lazily.
func (r *Registry) Get(symbol string) *engine.Engine {

	// Fast path (read lock)
	r.mu.RLock()
	e, exists := r.engines[symbol]
	r.mu.RUnlock()

	if exists {
		return e
	}

	// Slow path (write lock)
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if existing, ok := r.engines[symbol]; ok {
		return existing
	}

	e = engine.New(symbol)
	r.consumer.Start(
		context.Background(),
		e.Trades(),
	)

	r.engines[symbol] = e

	return e
}

// Exists checks whether an engine exists for a symbol.
func (r *Registry) Exists(symbol string) bool {

	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.engines[symbol]

	return exists
}

// Remove removes an engine from the registry.
func (r *Registry) Remove(symbol string) {

	r.mu.Lock()
	defer r.mu.Unlock()

	delete(
		r.engines,
		symbol,
	)
}

// Count returns the total number of engines.
func (r *Registry) Count() int {

	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.engines)
}

// Symbols returns all active symbols.
func (r *Registry) Symbols() []string {

	r.mu.RLock()
	defer r.mu.RUnlock()

	symbols := make(
		[]string,
		0,
		len(r.engines),
	)

	for symbol := range r.engines {
		symbols = append(
			symbols,
			symbol,
		)
	}

	return symbols
}
