package registry

import (
	"context"
	"sync"
	"time"

	"velocity/internal/engine"
	"velocity/internal/engine/snapshot"
	"velocity/internal/engine/wal"
	"velocity/internal/persistence/worker"
)

type Registry struct {
	engines map[string]*engine.Engine
	mu      sync.RWMutex

	consumer *worker.TradeConsumer

	snapshotWriter snapshot.SnapshotWriter
	walManager     *wal.Manager

	ctx    context.Context
	cancel context.CancelFunc
}

func New(
	snapshotWriter snapshot.SnapshotWriter,
	walManager *wal.Manager,
) *Registry {
	ctx, cancel := context.WithCancel(context.Background())

	return &Registry{
		engines: make(map[string]*engine.Engine),

		snapshotWriter: snapshotWriter,
		walManager:     walManager,

		ctx:    ctx,
		cancel: cancel,
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

	walWriter, err := r.walManager.Writer(symbol)
	if err != nil {
		return nil
	}

	e = engine.New(
		symbol,
		walWriter,
	)

	manager := snapshot.NewManager(
		r.snapshotWriter,
		30*time.Second,
		100000,
	)

	manager.Start(e)

	if r.consumer != nil {
		r.consumer.Start(
			r.ctx,
			e.Trades(),
		)
	}

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

	if e, ok := r.engines[symbol]; ok {
		e.Stop()
		delete(r.engines, symbol)
	}
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

// Shutdown stops every engine and cancels all trade-consumer goroutines
// started by this registry. Safe to call once, typically during
// application shutdown.
func (r *Registry) Shutdown() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cancel()

	for symbol, e := range r.engines {
		e.Stop()
		delete(r.engines, symbol)
	}

	if r.walManager != nil {
		return r.walManager.Close()
	}

	return nil
}

func (r *Registry) SetConsumer(
	consumer *worker.TradeConsumer,
) {
	r.consumer = consumer
}

func (r *Registry) Find(symbol string) (*engine.Engine, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.engines[symbol]
	return e, ok
}
