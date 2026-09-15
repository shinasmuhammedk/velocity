package registry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"velocity/internal/engine"
	"velocity/internal/engine/events"
	"velocity/internal/engine/snapshot"
	"velocity/internal/engine/wal"
	"velocity/internal/persistence/worker"
)

type Registry struct {
	engines map[string]*engine.Engine
	mu      sync.RWMutex

	// snapshotManagers mirrors `engines` by symbol - one entry per
	// engine, holding the periodic snapshot.Manager started for it in
	// Get(). Without this, Remove() and Shutdown() had no reference to
	// stop that manager's goroutine: it would keep ticking every 5
	// seconds forever, for every symbol ever loaded, holding a live
	// reference to the engine (so it could never be garbage collected
	// either) even after the registry believed it had been removed. See
	// test/stress/registry_churn_test.go.
	snapshotManagers map[string]*snapshot.Manager

	dispatcher *events.Dispatcher

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

		snapshotManagers: make(map[string]*snapshot.Manager),

		dispatcher: events.NewDispatcher(),

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
		r.dispatcher,
	)

	manager := snapshot.NewManager(
		r.snapshotWriter,
		5*time.Second,
		1,
	)

	manager.Start(e)

	r.snapshotManagers[symbol] = manager

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

	if manager, ok := r.snapshotManagers[symbol]; ok {
		manager.Stop()
		delete(r.snapshotManagers, symbol)
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

	for symbol, manager := range r.snapshotManagers {
		manager.Stop()
		delete(r.snapshotManagers, symbol)
	}

	if r.walManager != nil {
		return r.walManager.Close()
	}

	return nil
}

func (r *Registry) SetConsumer(
	consumer *worker.TradeConsumer,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.consumer = consumer

	for symbol, engine := range r.engines {
		fmt.Println("Starting consumer for:", symbol)
		consumer.Start(r.ctx, engine.Trades())
	}
}

func (r *Registry) Find(symbol string) (*engine.Engine, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.engines[symbol]
	return e, ok
}

func (r *Registry) Publisher() *events.Dispatcher {
	return r.dispatcher
}