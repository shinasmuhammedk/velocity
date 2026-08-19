package recovery

import (
	"errors"

	"velocity/internal/engine/registry"
	"velocity/internal/engine/snapshot"
	"velocity/internal/engine/wal"
)

type SnapshotRecovery struct {
	loader     *snapshot.Loader
	registry   *registry.Registry
	walManager *wal.Manager
}

func NewSnapshotRecovery(
	loader *snapshot.Loader,
	registry *registry.Registry,
	walManager *wal.Manager,
) *SnapshotRecovery {
	return &SnapshotRecovery{
		loader:     loader,
		registry:   registry,
		walManager: walManager,
	}
}

// Restore attempts to rebuild a symbol's order book from its last saved
// snapshot. Returns (true, nil) if a snapshot was found and applied,
// (false, nil) if no snapshot exists for this symbol (nothing to do —
// the caller should fall back to DB-based recovery for it), or
// (false, err) on a genuine read/apply failure.
func (r *SnapshotRecovery) Restore(symbol string) (bool, error) {
	snap, err := r.loader.Load(symbol)

	if err != nil {
		if errors.Is(err, snapshot.ErrSnapshotNotFound) {
			return false, nil
		}

		return false, err
	}

	engine := r.registry.Get(symbol)

	// Restore the snapshot first.
	engine.RestoreSnapshot(snap)
    // engine.SetSequence(snap.Sequence)

	// Open the WAL for this symbol.
	reader, err := r.walManager.Reader(symbol)
	if err != nil {
		return false, err
	}
	defer reader.Close()

	// Replay only events that happened after the snapshot.
	replayer := wal.NewReplayer(reader)

	events, err := replayer.Events(snap.Sequence)
	if err != nil {
		return false, err
	}

	// Apply the events to the restored engine state.
	applier := wal.NewApplier(
		engine.OrderBook(),
		engine.StopBook(),
	)

	for _, event := range events {
		if err := applier.Apply(event); err != nil {
			return false, err
		}
		engine.SetSequence(event.Sequence)
	}

	return true, nil
}
