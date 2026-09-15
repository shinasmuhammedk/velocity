package engine

import (
	"sync"
	"sync/atomic"
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

var submitResultPool = sync.Pool{
	New: func() any {
		return make(chan error, 1)
	},
}

func (e *Engine) start() {
	go func() {

		defer close(e.done)
		for cmd := range e.commandQueue {
			switch c := cmd; c.Kind {
			case command.Submit:

				if c.Order.Type == constants.StopMarketOrder ||
					c.Order.Type == constants.StopLimitOrder {

					e.stopBook.Add(c.Order)
					c.Order.Status = constants.OrderStatusPending

					e.incrementSequence()

					c.Result <- nil
					continue
				}

				seq := e.incrementSequence()

				if e.walWriter != nil {
					event := wal.NewSubmitEvent(
						seq,
						e.symbol,
						c.Order,
					)

					if err := e.walWriter.Write(event); err != nil {
						c.Result <- err
						continue
					}
				}

				trades, err := e.matcher.Match(c.Order)
				if err != nil {
					c.Result <- err
					continue
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

				seq := e.incrementSequence()

				if e.walWriter != nil {
					event := wal.NewCancelEvent(
						seq,
						e.symbol,
						c.OrderID,
					)

					if err := e.walWriter.Write(event); err != nil {
						c.Result <- err
						continue
					}
				}

				// First try stop orders.
				err := e.stopBook.CancelOrder(c.OrderID)

				if err == nil {
					c.Result <- nil
					continue
				}

				// Otherwise cancel from the normal order book.
				cancelledOrder, err := e.book.CancelOrder(c.OrderID)

				if err != nil {
					c.Result <- err
					continue
				}

				// Publish only after successful cancellation.
				e.publish(events.OrderCancelledEvent{
					BaseEvent: events.NewBaseEvent(),

					OrderID: cancelledOrder.ID,
					Symbol:  cancelledOrder.Symbol,
					UserID:  cancelledOrder.UserID,
				})

				c.Result <- nil

				// in engine.go's start()
			case command.Modify:

				seq := e.incrementSequence()

				if e.walWriter != nil {
					event := wal.NewModifyEvent(
						seq,
						e.symbol,
						c.OrderID,
						c.NewPrice,
						c.NewQuantity,
					)

					if err := e.walWriter.Write(event); err != nil {
						c.Result <- err
						continue
					}
				}

				err := e.book.ModifyOrder(
					c.OrderID,
					c.NewPrice,
					c.NewQuantity,
				)

				if err != nil {
					c.Result <- err
					continue
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
	}()
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

	// resultCh := make(chan error, 1)
	resultCh := submitResultPool.Get().(chan error)

	e.commandQueue <- command.Command{
		Kind:   command.Submit,
		Order:  order,
		Result: resultCh,
	}

	err := <-resultCh
	submitResultPool.Put(resultCh)

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
	resultCh := make(chan error, 1)
	e.commandQueue <- command.Command{
		Kind:    command.Cancel,
		OrderID: orderID,
		Result:  resultCh,
	}
	return <-resultCh // blocks until the background goroutine actually processes it
}

func (e *Engine) ModifyOrder(
	orderID int64,
	newPrice int64,
	newQuantity int64,
) error {

	resultCh := make(chan error, 1)

	e.commandQueue <- command.Command{
		Kind:        command.Modify,
		OrderID:     orderID,
		NewPrice:    newPrice,
		NewQuantity: newQuantity,
		Result:      resultCh,
	}

	return <-resultCh
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