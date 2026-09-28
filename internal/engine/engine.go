package engine

import (
	"sync"
	"sync/atomic"
	"time"
	"velocity/internal/domain/order"
	"velocity/internal/domain/trade"
	"velocity/internal/engine/command"
	"velocity/internal/engine/events"
	"velocity/internal/engine/matcher"
	"velocity/internal/engine/orderbook"
	"velocity/internal/engine/snapshot"
	"velocity/internal/engine/stopbook"
	"velocity/internal/engine/wal"
	"velocity/internal/infrastructure/metrics"
	"velocity/pkg/constants"
	"velocity/pkg/errors"
	"velocity/pkg/timeutil"
)

type Engine struct {
	symbol string

	book     *orderbook.OrderBook
	matcher  *matcher.Matcher
	stopBook *stopbook.StopBook

	publisher events.Publisher

	commandQueue chan command.Command
	tradeQueue   chan trade.Trade

	walWriter *wal.Writer

	lastTradePrice atomic.Int64
	sequence       atomic.Uint64

	// snapshotMu guards SnapshotState() against racing with a
	// recovery-time WAL replay (see ApplyReplayedEvent). Normal live
	// trading never touches it — commands are already serialized through
	// commandQueue — this exists solely to make "apply event, then
	// advance sequence" atomic with respect to a concurrently-running
	// periodic snapshot, since those two calls happen on different
	// goroutines with nothing else linking them.
	snapshotMu sync.Mutex

	done chan struct{} // new
}

// metricsSampleInterval is how often engine state gauges are refreshed.
// Kept comfortably below a typical 15s scrape interval so no scrape
// ever sees a stale sample.
const metricsSampleInterval = 5 * time.Second

var submitResultPool = sync.Pool{
	New: func() any {
		return make(chan error, 1)
	},
}

// maxCommandBatch caps how many already-queued commands are grouped
// behind a single WAL fsync.
//
// The cap exists for two reasons. A batch is held entirely in memory
// before it is written, so it bounds the append buffer; and every
// command in a batch shares the batch's fate, so it bounds how much
// work a single failed fsync throws away. 256 is large enough that the
// fsync cost per command is negligible at saturation and small enough
// that the buffer stays trivial.
const maxCommandBatch = 256

func (e *Engine) start() {
	go func() {
		defer close(e.done)

		// Both slices are allocated once and reused for every batch.
		batch := make([]command.Command, 0, maxCommandBatch)
		walEvents := make([]*wal.Event, 0, maxCommandBatch)

		for {
			// Block for the first command. This is what keeps group
			// commit from adding latency: a command that arrives on an
			// idle engine is not held waiting for company, it commits
			// immediately in a batch of one. Batches only grow when
			// commands are genuinely already queued behind a commit in
			// progress, which is exactly when amortizing the fsync is
			// free.
			first, ok := <-e.commandQueue
			if !ok {
				return
			}

			batch = append(batch[:0], first)
			closed := false

		drain:
			for len(batch) < maxCommandBatch {
				select {
				case next, ok := <-e.commandQueue:
					if !ok {
						// Stop() closed the queue. Finish this batch,
						// then exit.
						closed = true
						break drain
					}

					batch = append(batch, next)

				default:
					// Nothing else queued right now. Commit what we have
					// rather than waiting for more.
					break drain
				}
			}

			walEvents = e.processBatch(batch, walEvents[:0])

			if closed {
				return
			}
		}
	}()
}

// processBatch runs one group commit: assign sequence numbers, write the
// batch's WAL records with a single fsync, then apply each command to
// the book in arrival order.
//
// It returns the WAL event slice so the caller can reuse its backing
// array across batches.
func (e *Engine) processBatch(
	batch []command.Command,
	walEvents []*wal.Event,
) []*wal.Event {

	// Phase 1 — assign sequence numbers in arrival order and build the
	// WAL records.
	//
	// Every command consumes a sequence number here, including ones that
	// produce no WAL record, so the numbering is identical to what the
	// old one-command-at-a-time loop produced. Sequence must track
	// arrival order because snapshots use it to decide which WAL records
	// they already contain.
	for i := range batch {
		c := &batch[i]

		seq := e.incrementSequence()

		if e.walWriter == nil {
			continue
		}

		switch c.Kind {
		case command.Submit:
			// Stop orders rest in the stop book and are not written to
			// the WAL today; they consume a sequence number and nothing
			// more. (That they aren't durable is a pre-existing gap,
			// unchanged here.)
			if isStopOrder(c.Order) {
				continue
			}

			walEvents = append(walEvents, wal.NewSubmitEvent(
				seq,
				e.symbol,
				c.Order,
			))

		case command.Cancel:
			walEvents = append(walEvents, wal.NewCancelEvent(
				seq,
				e.symbol,
				c.OrderID,
			))

		case command.Modify:
			walEvents = append(walEvents, wal.NewModifyEvent(
				seq,
				e.symbol,
				c.OrderID,
				c.NewPrice,
				c.NewQuantity,
			))
		}
	}

	// Phase 2 — one append, one fsync, for the whole batch.
	if e.walWriter != nil && len(walEvents) > 0 {
		if err := e.walWriter.WriteBatch(walEvents); err != nil {
			// The batch is durable or it isn't, as a unit. Fail every
			// command in it and apply none of them, so the book never
			// diverges from what the WAL says happened.
			for i := range batch {
				batch[i].Result <- err
			}

			return walEvents
		}
	}

	// Phase 3 — apply. Everything below here is already durable.
	for i := range batch {
		e.applyCommand(&batch[i])
	}

	return walEvents
}

// applyCommand applies a single already-WAL'd command to the book and
// reports the outcome on its result channel.
func (e *Engine) applyCommand(c *command.Command) {
	switch c.Kind {

	case command.Submit:

		if isStopOrder(c.Order) {
			e.stopBook.Add(c.Order)
			c.Order.Status = constants.OrderStatusPending

			c.Result <- nil
			return
		}

		trades, err := e.matcher.Match(c.Order)
		if err != nil {
			c.Result <- err
			return
		}

		for _, t := range trades {

			e.lastTradePrice.Store(t.Price)
			metrics.TradesExecuted.Inc()

			e.tradeQueue <- t
			if e.publisher != nil {
				e.publish(events.TradeExecutedEvent{
					BaseEvent: events.NewBaseEvent(),

					TradeID: t.ID,

					BuyOrderID:  t.BuyOrderID,
					SellOrderID: t.SellOrderID,

					BuyerID:  t.BuyerID,
					SellerID: t.SellerID,

					Symbol: t.Symbol,

					Price:    t.Price,
					Quantity: t.Quantity,
				})
			}
		}

		e.processTriggeredStops()

		c.Result <- nil

	case command.Cancel:

		// First try stop orders.
		if err := e.stopBook.CancelOrder(c.OrderID); err == nil {
			c.Result <- nil
			return
		}

		// Otherwise cancel from the normal order book.
		cancelledOrder, err := e.book.CancelOrder(c.OrderID)

		if err != nil {
			c.Result <- err
			return
		}

		// Publish only after successful cancellation.
		e.publish(events.OrderCancelledEvent{
			BaseEvent: events.NewBaseEvent(),

			OrderID: cancelledOrder.ID,
			Symbol:  cancelledOrder.Symbol,
			UserID:  cancelledOrder.UserID,
		})

		c.Result <- nil

	case command.Modify:

		err := e.book.ModifyOrder(
			c.OrderID,
			c.NewPrice,
			c.NewQuantity,
		)

		if err != nil {
			c.Result <- err
			return
		}

		e.publish(events.OrderModifiedEvent{
			BaseEvent: events.NewBaseEvent(),

			OrderID:     c.OrderID,
			Symbol:      e.symbol,
			NewPrice:    c.NewPrice,
			NewQuantity: c.NewQuantity,
		})

		c.Result <- nil
	}
}

// isStopOrder reports whether an order rests in the stop book rather
// than the order book.
func isStopOrder(o *order.Order) bool {
	if o == nil {
		return false
	}

	return o.Type == constants.StopMarketOrder ||
		o.Type == constants.StopLimitOrder
}

func New(
	symbol string,
	walWriter *wal.Writer,
	publisher events.Publisher,
) *Engine {
	book := orderbook.New(symbol)

	e := &Engine{
		symbol:       symbol,
		walWriter:    walWriter,
		book:         book,
		matcher:      matcher.New(book),
		stopBook:     stopbook.New(),
		publisher:    publisher,
		commandQueue: make(chan command.Command, 100000),
		tradeQueue:   make(chan trade.Trade, 100000),
		done:         make(chan struct{}),
	}

	if walWriter != nil {
		e.SetSequence(walWriter.Sequence())
	}
	e.start()
	e.sampleMetrics(metricsSampleInterval)

	return e
}

func (e *Engine) SubmitOrder(
	order *order.Order,
) error {

	// POST_ONLY validation
	if order.TimeInForce == constants.TimeInForcePostOnly &&
		order.Type != constants.OrderTypeLimit {

		return errors.ErrPostOnlyMustBeLimit
	}

	// STOP order validation
	if order.Type == constants.StopMarketOrder ||
		order.Type == constants.StopLimitOrder {

		if order.StopPrice <= 0 {
			return errors.ErrInvalidStopPrice
		}

		if order.Side == constants.OrderSideBuy &&
			e.lastTradePrice.Load() > 0 &&
			order.StopPrice <= e.lastTradePrice.Load() {

			return errors.ErrBuyStopBelowMarket
		}

		if order.Side == constants.OrderSideSell &&
			e.lastTradePrice.Load() > 0 &&
			order.StopPrice >= e.lastTradePrice.Load() {

			return errors.ErrSellStopAboveMarket
		}
	}

	start := time.Now()

	// resultCh := make(chan error, 1)
	resultCh := submitResultPool.Get().(chan error)

	e.commandQueue <- command.Command{
		Kind:   command.Submit,
		Order:  order,
		Result: resultCh,
	}

	err := <-resultCh
	submitResultPool.Put(resultCh)

	e.observeCommand("submit", start, err)

	// return <-resultCh
	return err
}

// read-only channel accessor.
func (e *Engine) Trades() <-chan trade.Trade {
	return e.tradeQueue
}

func (e *Engine) OrderBook() *orderbook.OrderBook {
	return e.book
}

func (e *Engine) CancelOrder(orderID int64) error {
	start := time.Now()

	resultCh := make(chan error, 1)
	e.commandQueue <- command.Command{
		Kind:    command.Cancel,
		OrderID: orderID,
		Result:  resultCh,
	}

	// blocks until the background goroutine actually processes it
	err := <-resultCh

	e.observeCommand("cancel", start, err)

	return err
}

func (e *Engine) ModifyOrder(
	orderID int64,
	newPrice int64,
	newQuantity int64,
) error {

	start := time.Now()

	resultCh := make(chan error, 1)

	e.commandQueue <- command.Command{
		Kind:        command.Modify,
		OrderID:     orderID,
		NewPrice:    newPrice,
		NewQuantity: newQuantity,
		Result:      resultCh,
	}

	err := <-resultCh

	e.observeCommand("modify", start, err)

	return err
}

// observeCommand records end-to-end command latency and outcome.
//
// The measurement starts before the send on commandQueue, so a
// saturated queue shows up as latency here rather than as an invisible
// stall - which is the whole point of measuring at the caller boundary
// instead of inside the single-threaded loop.
func (e *Engine) observeCommand(
	kind string,
	start time.Time,
	err error,
) {
	outcome := "ok"
	if err != nil {
		outcome = "error"
	}

	metrics.EngineCommandsTotal.
		WithLabelValues(e.symbol, kind, outcome).
		Inc()

	metrics.EngineCommandDuration.
		WithLabelValues(e.symbol, kind).
		Observe(time.Since(start).Seconds())
}

// sampleMetrics periodically publishes gauge-style engine state.
//
// Gauges are sampled on a ticker rather than updated inline because
// they describe state, not events: writing them on every command would
// add label lookups to the hot path for no extra fidelity at a 15s
// scrape interval.
func (e *Engine) sampleMetrics(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		metrics.EngineQueueCapacity.
			WithLabelValues(e.symbol).
			Set(float64(cap(e.commandQueue)))

		for {
			select {
			case <-ticker.C:
				metrics.EngineQueueDepth.
					WithLabelValues(e.symbol).
					Set(float64(len(e.commandQueue)))

				metrics.EngineTradeQueueDepth.
					WithLabelValues(e.symbol).
					Set(float64(len(e.tradeQueue)))

				metrics.EngineSequence.
					WithLabelValues(e.symbol).
					Set(float64(e.Sequence()))

				bids, asks := e.book.SideCounts()

				metrics.OrderBookResting.
					WithLabelValues(e.symbol, "bid").
					Set(float64(bids))

				metrics.OrderBookResting.
					WithLabelValues(e.symbol, "ask").
					Set(float64(asks))

				metrics.OrderBookBestPrice.
					WithLabelValues(e.symbol, "bid").
					Set(float64(e.book.BestBidPrice()))

				metrics.OrderBookBestPrice.
					WithLabelValues(e.symbol, "ask").
					Set(float64(e.book.BestAskPrice()))

				metrics.StopBookResting.
					WithLabelValues(e.symbol).
					Set(float64(e.stopBook.Len()))

			case <-e.done:
				// The engine has stopped. Delete the series rather than
				// leaving them frozen at their last value: a removed
				// symbol whose gauges keep reporting a non-empty book
				// would read as stuck state forever.
				metrics.EngineQueueDepth.DeleteLabelValues(e.symbol)
				metrics.EngineQueueCapacity.DeleteLabelValues(e.symbol)
				metrics.EngineTradeQueueDepth.DeleteLabelValues(e.symbol)
				metrics.EngineSequence.DeleteLabelValues(e.symbol)
				metrics.StopBookResting.DeleteLabelValues(e.symbol)
				metrics.OrderBookResting.DeleteLabelValues(e.symbol, "bid")
				metrics.OrderBookResting.DeleteLabelValues(e.symbol, "ask")
				metrics.OrderBookBestPrice.DeleteLabelValues(e.symbol, "bid")
				metrics.OrderBookBestPrice.DeleteLabelValues(e.symbol, "ask")

				return
			}
		}
	}()
}

func (e *Engine) processTriggeredStops() {
	for {
		triggered := e.stopBook.Trigger(e.lastTradePrice.Load())

		if len(triggered) == 0 {
			return
		}

		for _, stopOrder := range triggered {

			e.incrementSequence()

			switch stopOrder.Type {

			case constants.StopMarketOrder:
				stopOrder.Type = constants.OrderTypeMarket
				stopOrder.StopPrice = 0

			case constants.StopLimitOrder:
				stopOrder.Type = constants.OrderTypeLimit
			}

			trades, err := e.matcher.Match(stopOrder)
			if err != nil {
				continue
			}

			for _, trade := range trades {
				e.lastTradePrice.Store(trade.Price)
				metrics.TradesExecuted.Inc()
				e.tradeQueue <- trade
			}
		}
	}
}

func (e *Engine) LastTradePrice() int64 {
	return e.lastTradePrice.Load()
}

func (e *Engine) StopBook() *stopbook.StopBook {
	return e.stopBook
}

func (e *Engine) Stop() {
	close(e.commandQueue)
	<-e.done
}

func (e *Engine) RecoverOrder(o *order.Order) {

	switch o.Type {

	case constants.StopMarketOrder,
		constants.StopLimitOrder:

		e.stopBook.Add(o)

	default:

		e.book.AddOrder(o)
	}
}

func (e *Engine) ReplayWAL(reader *wal.Reader) error {
	replayer := wal.NewReplayer(reader)

	events, err := replayer.Events(e.Sequence())
	if err != nil {
		return err
	}

	applier := wal.NewApplier(
		e.book,
		e.stopBook,
	)

	for _, event := range events {
		if err := applier.Apply(event); err != nil {
			return err
		}

		e.SetSequence(event.Sequence)

		if event.Order != nil {
			e.SetLastTradePrice(e.LastTradePrice())
		}
	}

	return nil
}

func (e *Engine) Sequence() uint64 {
	return e.sequence.Load()
}

func (e *Engine) SetSequence(seq uint64) {
	e.sequence.Store(seq)
}

func (e *Engine) SetLastTradePrice(price int64) {
	e.lastTradePrice.Store(price)
}

func (e *Engine) incrementSequence() uint64 {
	return e.sequence.Add(1)
}

func (e *Engine) SnapshotState() *snapshot.Snapshot {
	// Held for the whole read so this can never observe a state where an
	// event has been applied to the book but the sequence counter hasn't
	// been advanced to match yet (see ApplyReplayedEvent) - that
	// half-applied window is exactly what let a replayed event get
	// double-counted after the next recovery.
	e.snapshotMu.Lock()
	defer e.snapshotMu.Unlock()

	return &snapshot.Snapshot{
		Symbol:         e.symbol,
		Sequence:       e.Sequence(),
		LastTradePrice: e.LastTradePrice(),
		ActiveOrders:   e.book.ActiveOrders(),
		StopOrders:     e.stopBook.Orders(),
		CreatedAt:      timeutil.UTCNow(),
	}
}

// ApplyReplayedEvent applies a single WAL event to the engine's book during
// recovery and advances the sequence counter to match, as one atomic step.
// SnapshotRecovery.Restore must use this - not applier.Apply followed by a
// separate SetSequence call - because the periodic snapshot goroutine
// (started the moment registry.Get creates this engine, possibly before
// recovery finishes) can otherwise capture a snapshot in the gap between
// the two: one whose ActiveOrders already reflects the event but whose
// declared Sequence hasn't caught up. That snapshot understates its own
// sequence, so the same event gets replayed - and reapplied - on the next
// recovery. See test/chaos/snapshot_race_test.go for the full failure mode.
func (e *Engine) ApplyReplayedEvent(
	applier *wal.Applier,
	event *wal.Event,
) error {
	e.snapshotMu.Lock()
	defer e.snapshotMu.Unlock()

	if err := applier.Apply(event); err != nil {
		return err
	}

	e.SetSequence(event.Sequence)

	return nil
}

func (e *Engine) RestoreSnapshot(
	s *snapshot.Snapshot,
) {
	e.sequence.Store(s.Sequence)

	e.lastTradePrice.Store(
		s.LastTradePrice,
	)

	for _, o := range s.ActiveOrders {
		e.book.AddOrder(o)
	}

	for _, o := range s.StopOrders {
		e.stopBook.Add(o)
	}
}

func (e *Engine) publish(event events.Event) {
	if e.publisher != nil {
		e.publisher.Publish(event)
	}
}