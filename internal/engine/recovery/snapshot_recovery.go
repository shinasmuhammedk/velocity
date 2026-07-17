package recovery

import (
	"errors"

	"velocity/internal/engine/registry"
	"velocity/internal/engine/snapshot"
)

type SnapshotRecovery struct {
	loader   *snapshot.Loader
	registry *registry.Registry
}

func NewSnapshotRecovery(
	loader *snapshot.Loader,
	registry *registry.Registry,
) *SnapshotRecovery {
	return &SnapshotRecovery{
		loader:   loader,
		registry: registry,
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

	engine.RestoreSnapshot(snap)

	return true, nil
}