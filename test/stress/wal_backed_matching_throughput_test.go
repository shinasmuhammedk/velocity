package stress

// ---------------------------------------------------------------------------
// What this file is testing
//
// wal_backed_throughput_test.go measures the WAL's fsync cost on a
// non-crossing workload (one SUBMIT event per order, no matching). That
// was flagged in docs/performance/benchmark-results.md as the cheapest
// possible per-order WAL cost, with matching-heavy scenarios explicitly
// called out as unmeasured: matching produces additional WAL events per
// trade (fills on both sides), so the gap for a matching-heavy workload
// was expected to differ and worth measuring separately.
//
// This is that measurement. Same two-phase structure as
// wal_backed_throughput_test.go (no WAL, then a real wal.Writer against
// a temp dir on disk, both in the same process on the same machine), but
// with test/load's 100%-crossing workload: every incoming BUY matches
// immediately against a large resting SELL, so every submitted order
// produces a trade and (with a real WAL) at least one extra WAL write
// for the fill, on top of the SUBMIT write the non-crossing case has.
//
// Same caveats apply as the non-crossing test: fsync cost is disk- and
// filesystem-dependent, this is a single run on a single machine, and
// this does not fail on a throughput regression for the same reason -
// there's no portable threshold that holds across every disk this suite
// could run on. It fails only on submit errors or zero orders processed.
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

func matchingStressBuyOrder(id int64) *order.Order {
	return &order.Order{
		ID:          id,
		UserID:      3001,
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    1,
		Remaining:   1,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}
}

func matchingStressSellOrder(id int64) *order.Order {
	return &order.Order{
		ID:          id,
		UserID:      3002,
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

// runMatchingThroughputPhase submits 100%-crossing BUY orders from
// `producers` concurrent goroutines for `duration` against a large
// resting SELL seeded on `e` beforehand, and returns the achieved
// orders/sec and trades/sec. Mirrors test/load's
// TestEngineLoad_Matching_4Producers_30Seconds workload exactly, so the
// numbers are comparable to what that test reports.
func runMatchingThroughputPhase(
	t *testing.T,
	e *engine.Engine,
	producers int,
	duration time.Duration,
	idBase int64,
) (orderThroughput, tradeThroughput float64, submitted, trades, failed uint64) {
	t.Helper()

	// Large resting liquidity - every incoming BUY at 1000 matches
	// against this SELL.
	seed := matchingStressSellOrder(idBase - 1)
	seed.Quantity = 1_000_000_000
	seed.Remaining = seed.Quantity

	require.NoError(t, e.SubmitOrder(seed))
	require.Equal(
		t, int64(1000), e.OrderBook().BestAskPrice(),
		"seed order did not enter the book",
	)

	var submittedCount atomic.Uint64
	var tradeCount atomic.Uint64
	var errorCount atomic.Uint64

	// Exactly ONE consumer drains and counts all trades - multiple
	// consumers would distribute channel messages between them and
	// make the trade count incorrect.
	stopDrain := make(chan struct{})
	drainDone := make(chan struct{})

	go func() {
		defer close(drainDone)
		for {
			select {
			case <-e.Trades():
				tradeCount.Add(1)
			case <-stopDrain:
				return
			}
		}
	}()

	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(producers)

	for p := 0; p < producers; p++ {
		go func(producerID int) {
			defer wg.Done()

			localID := idBase + int64(producerID)*1_000_000_000

			for time.Since(start) < duration {
				buy := matchingStressBuyOrder(localID)
				localID += int64(producers)

				if err := e.SubmitOrder(buy); err != nil {
					errorCount.Add(1)
					continue
				}

				submittedCount.Add(1)
			}
		}(p)
	}

	wg.Wait()

	elapsed := time.Since(start)
	totalOrders := submittedCount.Load()

	// SubmitOrder is asynchronous with respect to the engine worker, so
	// producers finishing does not mean all trades have been emitted
	// yet - wait for the trade count to catch up before we stop
	// draining.
	require.Eventually(
		t,
		func() bool {
			return tradeCount.Load() == totalOrders
		},
		5*time.Second,
		10*time.Millisecond,
		"engine did not produce a trade for every submitted order",
	)

	close(stopDrain)
	<-drainDone

	submitted = totalOrders
	trades = tradeCount.Load()
	failed = errorCount.Load()
	orderThroughput = float64(submitted) / elapsed.Seconds()
	tradeThroughput = float64(trades) / elapsed.Seconds()

	return orderThroughput, tradeThroughput, submitted, trades, failed
}

func TestStress_SustainedMatchingThroughput_WithAndWithoutRealWAL(t *testing.T) {
	const (
		producers = 4
		duration  = 10 * time.Second
	)

	// ------------------------------------------------------------------
	// Phase 1: no WAL - identical to what
	// TestEngineLoad_Matching_4Producers_30Seconds in test/load already
	// measures (just a shorter duration here, to keep this test fast).
	// ------------------------------------------------------------------

	noWALEngine := engine.New("BTCUSDT", nil, nil)

	noWALOrderTput, noWALTradeTput, noWALSubmitted, noWALTrades, noWALFailed :=
		runMatchingThroughputPhase(t, noWALEngine, producers, duration, 30_000_000_000)
	noWALEngine.Stop()

	require.Zero(t, noWALFailed, "no-WAL phase produced %d submit errors", noWALFailed)
	require.NotZero(t, noWALSubmitted, "no-WAL phase submitted zero orders")

	// ------------------------------------------------------------------
	// Phase 2: identical crossing workload, but with a real wal.Writer
	// backed by an actual file on disk, fsync included on every write -
	// exactly what registry.Get() attaches in production. Matching
	// produces a WAL write for the incoming order plus fill events, so
	// this phase does strictly more disk I/O per order than the
	// non-crossing WAL test.
	// ------------------------------------------------------------------

	dir := t.TempDir()
	walManager := wal.NewManager(dir, wal.NewJSONSerializer())
	walWriter, err := walManager.Writer("BTCUSDT")
	require.NoError(t, err)

	walEngine := engine.New("BTCUSDT", walWriter, nil)

	walOrderTput, walTradeTput, walSubmitted, walTrades, walFailed :=
		runMatchingThroughputPhase(t, walEngine, producers, duration, 40_000_000_000)
	walEngine.Stop()
	require.NoError(t, walManager.Close())

	require.Zero(t, walFailed, "WAL-backed phase produced %d submit errors", walFailed)
	require.NotZero(t, walSubmitted, "WAL-backed phase submitted zero orders")

	// ------------------------------------------------------------------
	// Report both numbers side by side, and against the non-crossing
	// WAL-backed figure this repo already measured, so all three are
	// visible together.
	// ------------------------------------------------------------------

	slowdown := noWALOrderTput / walOrderTput

	t.Logf("========================================================")
	t.Logf("Velocity Sustained Matching Throughput: WAL cost")
	t.Logf("========================================================")
	t.Logf("workload:               100%% crossing BUY")
	t.Logf("producers:              %d", producers)
	t.Logf("duration per phase:     %s", duration)
	t.Logf("--------------------------------------------------------")
	t.Logf("no WAL   orders/sec:    %.2f  (submitted=%d, trades=%d)", noWALOrderTput, noWALSubmitted, noWALTrades)
	t.Logf("real WAL orders/sec:    %.2f  (submitted=%d, trades=%d)", walOrderTput, walSubmitted, walTrades)
	t.Logf("no WAL   trades/sec:    %.2f", noWALTradeTput)
	t.Logf("real WAL trades/sec:    %.2f", walTradeTput)
	t.Logf("--------------------------------------------------------")
	t.Logf("fsync-per-write slowdown factor: %.1fx", slowdown)
	t.Logf("========================================================")
	t.Logf("Compare against TestStress_SustainedThroughput_WithAndWithoutRealWAL's")
	t.Logf("non-crossing WAL-backed figure: matching does strictly more WAL")
	t.Logf("writes per order (fills on top of the SUBMIT event), so this")
	t.Logf("number - not the non-crossing one - is what matters for capacity")
	t.Logf("planning on a symbol with real crossing activity.")
	t.Logf("========================================================")

	fmt.Printf(
		"\nVelocity matching WAL-backed throughput: %.2f orders/sec (%.1fx slower than no-WAL)\n\n",
		walOrderTput, slowdown,
	)
}