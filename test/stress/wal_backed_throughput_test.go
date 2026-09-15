package stress

// ---------------------------------------------------------------------------
// What this file is testing
//
// Every existing throughput number in this repo - TestEngineLoad_8Producers
// _30Seconds and its siblings in test/load - calls engine.New(symbol, nil,
// nil). No WAL writer, no publisher. That measures pure in-memory matching:
// producers -> command queue -> single engine goroutine -> order book, with
// zero disk I/O anywhere in the path.
//
// Production never runs that configuration. registry.Get() always attaches
// a real wal.Writer, and wal.Writer.Write() calls file.Write() followed by
// file.Sync() on every single command (internal/engine/wal/writer.go) -
// full fsync, not a buffered/batched write. SubmitOrder is also synchronous
// per call: it blocks on the command's result channel until the engine's
// single worker goroutine has fully processed it, WAL write included
// (internal/engine/engine.go SubmitOrder). So sustained throughput in
// production is bounded by however fast that one goroutine can push
// through fsync, not by CPU or by how many producer goroutines are
// hammering it concurrently.
//
// This test runs the identical non-crossing workload twice, back to back,
// in the same process: once exactly as test/load does (no WAL), once with
// a real wal.Writer pointed at a temp directory on this machine's disk.
// The point isn't a specific number - fsync cost varies enormously by
// disk, filesystem, and OS - it's putting the two numbers side by side so
// the gap between "what our benchmarks report" and "what production can
// actually sustain" is visible and gets checked over time, not just
// asserted once and forgotten.
//
// This does not fail on a throughput regression, because there's no
// portable threshold that holds across every disk this suite could run
// on. It fails only on the same terms as the existing load tests: submit
// errors, or zero orders processed.
// ---------------------------------------------------------------------------

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"velocity/internal/domain/order"
	"velocity/internal/engine"
	"velocity/internal/engine/wal"
	"velocity/pkg/constants"

	"github.com/stretchr/testify/require"
)

func stressSellOrder(id int64) *order.Order {
	return &order.Order{
		ID:          id,
		UserID:      2001,
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideSell,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    1,
		Remaining:   1,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}
}

// runThroughputPhase submits non-crossing SELL orders from `producers`
// concurrent goroutines for `duration`, and returns the achieved
// orders/sec. Mirrors test/load's TestEngineLoad_8Producers_30Seconds
// workload exactly, so the two numbers are comparable.
func runThroughputPhase(
	t *testing.T,
	e *engine.Engine,
	producers int,
	duration time.Duration,
	idBase int64,
) (throughput float64, submitted uint64, failed uint64) {
	t.Helper()

	stopDrain := make(chan struct{})
	drainDone := make(chan struct{})

	go func() {
		defer close(drainDone)
		for {
			select {
			case <-e.Trades():
			case <-stopDrain:
				return
			}
		}
	}()

	var submittedCount atomic.Uint64
	var errorCount atomic.Uint64

	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(producers)

	for p := 0; p < producers; p++ {
		go func(producerID int) {
			defer wg.Done()

			localID := idBase + int64(producerID)*1_000_000_000

			for time.Since(start) < duration {
				o := stressSellOrder(localID)
				localID += int64(producers)

				if err := e.SubmitOrder(o); err != nil {
					errorCount.Add(1)
					continue
				}

				submittedCount.Add(1)
			}
		}(p)
	}

	wg.Wait()

	elapsed := time.Since(start)

	close(stopDrain)
	<-drainDone

	submitted = submittedCount.Load()
	failed = errorCount.Load()
	throughput = float64(submitted) / elapsed.Seconds()

	return throughput, submitted, failed
}

func TestStress_SustainedThroughput_WithAndWithoutRealWAL(t *testing.T) {
	const (
		producers = 8
		duration  = 10 * time.Second
	)

	// ------------------------------------------------------------------
	// Phase 1: no WAL at all - identical to what test/load already
	// measures. Included here so both numbers come from one run on one
	// machine, rather than asking anyone to compare against a number
	// from a different test file run at a different time.
	// ------------------------------------------------------------------

	noWALEngine := engine.New("BTCUSDT", nil, nil)

	noWALThroughput, noWALSubmitted, noWALFailed := runThroughputPhase(
		t, noWALEngine, producers, duration, 10_000_000_000,
	)
	noWALEngine.Stop()

	require.Zero(t, noWALFailed, "no-WAL phase produced %d submit errors", noWALFailed)
	require.NotZero(t, noWALSubmitted, "no-WAL phase submitted zero orders")

	// ------------------------------------------------------------------
	// Phase 2: identical workload, but with a real wal.Writer backed by
	// an actual file on disk, fsync included on every write - exactly
	// what registry.Get() attaches in production.
	// ------------------------------------------------------------------

	dir := t.TempDir()
	walManager := wal.NewManager(dir, wal.NewJSONSerializer())
	walWriter, err := walManager.Writer("BTCUSDT")
	require.NoError(t, err)

	walEngine := engine.New("BTCUSDT", walWriter, nil)

	walThroughput, walSubmitted, walFailed := runThroughputPhase(
		t, walEngine, producers, duration, 20_000_000_000,
	)
	walEngine.Stop()
	require.NoError(t, walManager.Close())

	require.Zero(t, walFailed, "WAL-backed phase produced %d submit errors", walFailed)
	require.NotZero(t, walSubmitted, "WAL-backed phase submitted zero orders")

	// ------------------------------------------------------------------
	// Report both numbers side by side. This is the actual point of the
	// test - making the gap visible on every run, not a one-time
	// assertion.
	// ------------------------------------------------------------------

	slowdown := noWALThroughput / walThroughput

	t.Logf("========================================================")
	t.Logf("Velocity Sustained Throughput: WAL cost")
	t.Logf("========================================================")
	t.Logf("producers:              %d", producers)
	t.Logf("duration per phase:     %s", duration)
	t.Logf("--------------------------------------------------------")
	t.Logf("no WAL   orders/sec:    %.2f  (submitted=%d)", noWALThroughput, noWALSubmitted)
	t.Logf("real WAL orders/sec:    %.2f  (submitted=%d)", walThroughput, walSubmitted)
	t.Logf("--------------------------------------------------------")
	t.Logf("fsync-per-write slowdown factor: %.1fx", slowdown)
	t.Logf("========================================================")
	t.Logf("Every number test/load reports is the no-WAL figure above.")
	t.Logf("Production always runs with a real WAL writer attached")
	t.Logf("(registry.Get), so the WAL-backed figure - not the no-WAL")
	t.Logf("one - is the number that reflects what this engine can")
	t.Logf("actually sustain per symbol in production.")
	t.Logf("========================================================")

	fmt.Printf(
		"\nVelocity WAL-backed throughput: %.2f orders/sec (%.1fx slower than no-WAL)\n\n",
		walThroughput, slowdown,
	)
}
