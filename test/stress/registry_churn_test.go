package stress

// ---------------------------------------------------------------------------
// What this file is testing
//
// registry.Get() (internal/engine/registry/registry.go) does this on every
// lazy engine creation:
//
//	manager := snapshot.NewManager(r.snapshotWriter, 5*time.Second, 1)
//	manager.Start(e)
//
// `manager` is a local variable. It is never stored on the Registry, never
// stored on the Engine, never returned to the caller - there is no field
// anywhere that holds a reference to it once Get() returns. manager.Start
// spawns a goroutine that ticks every 5 seconds for the rest of the
// process's life, closing over `e` (the Snapshotable target).
//
// Registry.Remove(symbol) calls e.Stop() and deletes the map entry.
// Registry.Shutdown() does the same for every engine. Neither has any
// reference to that symbol's snapshot.Manager, so neither can call its
// Stop() method - which is the only thing that would make its goroutine
// exit (see snapshot/manager.go: the ticker loop only returns on <-m.stop).
//
// Two things leak together, permanently, for the life of the process:
//
//  1. The goroutine itself - it keeps ticking every 5 seconds forever,
//     for every symbol ever loaded, even ones long since removed.
//  2. The Engine it closes over - since the leaked goroutine holds a live
//     reference to `e` (to call e.Sequence()/e.SnapshotState() on each
//     tick), the garbage collector can never reclaim that engine's order
//     book, stop book, or any order data it holds, even after
//     Registry.Remove or Registry.Shutdown believes it's gone.
//
// This doesn't show up in a quick smoke test - one extra goroutine is
// invisible. It shows up under sustained operation: a long-running
// matchnode process that loads many symbols over its lifetime (or a
// service that creates/tears down registries repeatedly, as tests and
// some deployment patterns do) accumulates one leaked goroutine and one
// pinned engine per symbol, forever. That's exactly what this stress test
// simulates: many registry lifecycles, many symbols each, checking that
// goroutine count returns to baseline after every single one.
// ---------------------------------------------------------------------------

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"velocity/internal/engine/registry"
	"velocity/internal/engine/snapshot"
	"velocity/internal/engine/wal"

	"github.com/stretchr/testify/require"
)

// stableGoroutineCount forces a GC pass and gives already-finished
// goroutines a moment to actually unwind before sampling
// runtime.NumGoroutine() - a leaked goroutine survives this; a merely
// slow-to-exit one does not, which is the whole point of the distinction.
func stableGoroutineCount(t *testing.T) int {
	t.Helper()

	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	return runtime.NumGoroutine()
}

func TestStress_RegistryChurn_SnapshotManagerGoroutinesMustNotLeak(t *testing.T) {
	const (
		cycles          = 20
		symbolsPerCycle = 10
	)

	baseline := stableGoroutineCount(t)

	for cycle := 0; cycle < cycles; cycle++ {

		dir := t.TempDir()
		walManager := wal.NewManager(dir, wal.NewJSONSerializer())
		r := registry.New(&snapshot.MockWriter{}, walManager)

		for i := 0; i < symbolsPerCycle; i++ {
			symbol := fmt.Sprintf("SYM-%d-%d", cycle, i)

			e := r.Get(symbol)
			require.NotNilf(t, e, "engine creation failed for %s", symbol)
		}

		// From the caller's point of view, this is a complete, clean
		// shutdown - every engine's own goroutine really does stop
		// (Registry.Shutdown calls e.Stop() on each, which blocks until
		// that engine's command-processing goroutine has exited). What
		// it can't touch is the snapshot manager goroutine it never
		// kept a reference to.
		require.NoError(t, r.Shutdown())
	}

	after := stableGoroutineCount(t)

	totalEnginesCreated := cycles * symbolsPerCycle

	// A handful of goroutines of slack for the Go runtime's own
	// background workers (GC, etc.), which fluctuate independently of
	// anything this test does. A leak here isn't off by two or three -
	// it's off by exactly the number of engines created, because every
	// single one leaks its own snapshot-manager goroutine.
	const slack = 5

	require.LessOrEqualf(t, after, baseline+slack,
		"BUG: goroutine count went from %d to %d (+%d) after %d complete "+
			"registry lifecycles totalling %d engines, all of which were "+
			"cleanly Shutdown(). Each registry.Get() starts a periodic "+
			"snapshot.Manager goroutine that is never stored anywhere, so "+
			"Registry.Shutdown() (and Registry.Remove()) has no reference "+
			"to stop it. Those goroutines - and the engines they hold a "+
			"live reference to, which can then never be garbage collected "+
			"- leak for the remaining life of the process.",
		baseline, after, after-baseline, cycles, totalEnginesCreated)
}