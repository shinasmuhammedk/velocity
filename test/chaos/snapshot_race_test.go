package chaos_test

// ---------------------------------------------------------------------------
// What this file is testing
//
// registry.Get() does two things when it lazily creates an engine:
//
//   1. engine.New(...) - which starts the engine's own single-writer
//      goroutine (e.start()).
//   2. snapshot.NewManager(...).Start(e) - which starts a SEPARATE
//      goroutine that, every 5 seconds, calls target.Sequence() and
//      target.SnapshotState() directly on that same engine.
//
// Both of those happen the instant the engine is created - including
// during startup, before SnapshotRecovery.Restore() has finished replaying
// that symbol's post-snapshot WAL events into it.
//
// SnapshotRecovery.Restore() (internal/engine/recovery/snapshot_recovery.go)
// replays each event like this:
//
//	applier.Apply(event)          // mutates engine.OrderBook() / StopBook()
//	engine.SetSequence(event.Sequence)   // <-- happens AFTER, not atomically
//
// There is no lock spanning those two calls, and nothing prevents the
// periodic snapshot goroutine from calling SnapshotState()/Sequence() in
// the gap between them. If it does, it captures a snapshot whose
// ActiveOrders already reflects the just-applied event, but whose declared
// Sequence has not been advanced to match - a snapshot that lies about how
// much of the WAL it represents.
//
// That lie has a concrete, financial consequence: SnapshotRecovery.Restore
// resumes WAL replay from *snap.Sequence + 1* on the next recovery. Since
// the bad snapshot understates its own sequence, the event it already
// contains gets replayed again - and orderbook.AddOrder has no duplicate-ID
// guard (see internal/engine/orderbook/order_book.go addOrderWithoutLock:
// it unconditionally appends to the price level's FIFO queue and simply
// overwrites the Orders[id] map entry). The result is a single order
// physically resting TWICE in the matching queue.
//
// It gets worse on cancel. CancelOrder(id) looks the order up through the
// Orders map - which, after the overwrite above, points to only ONE of the
// two physical FIFO entries - and PriceLevel.Remove matches by ID and
// stops at the first hit (order_book.go CancelOrder, price_level.go
// Remove). So a cancel that the caller and the database both record as
// successful leaves the OTHER copy resting in the book: live, matchable,
// and permanently unreachable by ID, since Orders[id] no longer exists to
// find it again.
//
// Test 1 forces the exact interleaving deterministically (via channels,
// not real timers - relying on the real 5s ticker to land in a few-
// microsecond window would make this un-reproducible in CI) and shows the
// snapshot is self-inconsistent the moment it's captured.
//
// Test 2 carries that bad snapshot through the real recovery pipeline a
// second time, confirms the duplicate lands in the live book, and then
// cancels it to show the zombie copy that's left behind.
// ---------------------------------------------------------------------------

import (
	"testing"

	"velocity/internal/domain/order"
	"velocity/internal/engine/orderbook"
	"velocity/internal/engine/registry"
	"velocity/internal/engine/recovery"
	"velocity/internal/engine/snapshot"
	"velocity/internal/engine/wal"
	"velocity/pkg/constants"

	"github.com/stretchr/testify/require"
)

func testOrder(symbol string) *order.Order {
	return &order.Order{
		ID:          1,
		UserID:      1,
		Symbol:      symbol,
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		TimeInForce: constants.TimeInForceGTC,
		Status:      constants.OrderStatusOpen,
		Price:       100,
		Quantity:    1,
		Remaining:   1,
	}
}

// TestChaos_SnapshotDuringReplay_CapturesInconsistentSnapshot forces a
// snapshot capture to land in the gap between "event applied to the book"
// and "sequence advanced to match" during WAL replay, and shows the
// resulting snapshot is self-contradictory.
func TestChaos_SnapshotDuringReplay_CapturesInconsistentSnapshot(t *testing.T) {
	dir := t.TempDir()
	symbol := "BTCUSDT"

	walManager := wal.NewManager(dir, wal.NewJSONSerializer())
	r := registry.New(&snapshot.MockWriter{}, walManager)
	defer r.Shutdown()

	e := r.Get(symbol)
	require.NotNil(t, e)

	event := wal.NewSubmitEvent(1, symbol, testOrder(symbol))
	applier := wal.NewApplier(e.OrderBook(), e.StopBook())

	applied := make(chan struct{})
	proceed := make(chan struct{})
	snapCh := make(chan *snapshot.Snapshot, 1)

	// Stand-in for the periodic snapshot.Manager goroutine: it calls the
	// exact same public methods (SnapshotState) the real ticker-driven
	// code calls, with no coordination with whoever is mutating the
	// engine concurrently - because in production there isn't any.
	go func() {
		<-applied
		snapCh <- e.SnapshotState()
		close(proceed)
	}()

	// One iteration of SnapshotRecovery.Restore()'s replay loop, in the
	// same order the real code runs it: apply, THEN advance sequence.
	applier.Apply(event)
	close(applied)
	<-proceed
	e.SetSequence(event.Sequence)

	snap := <-snapCh

	require.Lenf(t, snap.ActiveOrders, 1,
		"expected the racing snapshot to already contain the order "+
			"applied moments earlier, got %d active orders",
		len(snap.ActiveOrders))

	require.Lessf(t, snap.Sequence, event.Sequence,
		"snapshot declares sequence %d while its own ActiveOrders already "+
			"contains the effects of event %d - this snapshot understates "+
			"itself, which is exactly what causes event %d to be replayed "+
			"a second time on the next recovery",
		snap.Sequence, event.Sequence, event.Sequence)
}

// TestChaos_SnapshotRaceDuringReplay_MustNotDuplicateOrderOnNextRecovery
// carries a snapshot produced by that exact race through the real
// SnapshotRecovery pipeline a second time - simulating a crash shortly
// after the bad snapshot was written - and checks the resulting order book
// for a duplicated resting order.
//
// This is expected to FAIL against the current code: it is the concrete,
// money-relevant consequence of the race demonstrated above.
func TestChaos_SnapshotRaceDuringReplay_MustNotDuplicateOrderOnNextRecovery(t *testing.T) {
	dir := t.TempDir()
	symbol := "BTCUSDT"
	serializer := wal.NewJSONSerializer()

	// ------------------------------------------------------------------
	// Step 1: write the real WAL file a live engine would have produced -
	// one SUBMIT event at sequence 1.
	// ------------------------------------------------------------------

	walManager := wal.NewManager(dir, serializer)
	writer, err := walManager.Writer(symbol)
	require.NoError(t, err)

	event := wal.NewSubmitEvent(1, symbol, testOrder(symbol))
	require.NoError(t, writer.Write(event))
	require.NoError(t, walManager.Close())

	// ------------------------------------------------------------------
	// Step 2: produce the bad snapshot exactly as the race in Test 1
	// does - sequence 0, but ActiveOrders already containing order 1 -
	// and write it to disk as SnapshotRecovery would load it.
	// ------------------------------------------------------------------

	// Build the book a live engine would have at the moment the racing
	// snapshot goroutine reads it: order 1 already resting, sequence
	// still at the pre-event value of 0.
	book := orderbook.New(symbol)
	book.AddOrder(testOrder(symbol))

	badSnapshot := &snapshot.Snapshot{
		Symbol:       symbol,
		Sequence:     0, // stale: understates what ActiveOrders holds
		ActiveOrders: book.ActiveOrders(),
	}

	snapWriter := snapshot.NewWriter(dir, snapshot.NewJSONSerializer())
	require.NoError(t, snapWriter.Write(badSnapshot))

	// ------------------------------------------------------------------
	// Step 3: simulate a restart. Fresh registry, fresh WAL manager
	// pointed at the same directory, real SnapshotRecovery.Restore -
	// exactly the startup path in internal/app/bootstrap.go.
	// ------------------------------------------------------------------

	walManager2 := wal.NewManager(dir, serializer)
	r := registry.New(&snapshot.MockWriter{}, walManager2)
	defer r.Shutdown()

	loader := snapshot.NewLoader(dir, snapshot.NewJSONSerializer())
	snapshotRecovery := recovery.NewSnapshotRecovery(loader, r, walManager2)

	restored, err := snapshotRecovery.Restore(symbol)
	require.NoError(t, err)
	require.True(t, restored)

	e := r.Get(symbol)
	require.NotNil(t, e)

	// ------------------------------------------------------------------
	// The bug: order 1 is now resting twice in the same price level's
	// FIFO queue.
	// ------------------------------------------------------------------

	level, ok := e.OrderBook().Bids[100]
	require.True(t, ok, "expected a bid price level at 100")

	require.Equalf(t, 1, level.Size(),
		"BUG: order 1 is resting %d times in the price-level FIFO queue "+
			"at price 100 - the snapshot-during-replay race caused its "+
			"SUBMIT event to be applied twice, and orderbook.AddOrder has "+
			"no duplicate-ID guard",
		level.Size())

	// ------------------------------------------------------------------
	// The consequence: cancelling order 1 looks up Orders[1] in the
	// ID-keyed map - which points to only ONE of the two physical FIFO
	// entries - and PriceLevel.Remove matches by ID and stops at the
	// first hit. So a cancel that the caller (and the database) will
	// record as successful leaves the other copy resting in the book
	// forever: live, matchable, and no longer reachable by ID at all.
	// ------------------------------------------------------------------

	_, err = e.OrderBook().CancelOrder(1)
	require.NoError(t, err, "cancel reports success, as it always would here")

	require.Equalf(t, 0, level.Size(),
		"BUG: after cancelling order 1, %d copies are still resting at "+
			"price 100 - the user (and the database) believe this order "+
			"is cancelled, but a live, matchable duplicate remains in the "+
			"book with no order ID pointing to it, and can never be "+
			"cancelled again",
		level.Size())
}