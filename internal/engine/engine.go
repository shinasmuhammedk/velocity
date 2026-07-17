package engine

import (
	"sync/atomic"
	"velocity/internal/domain/order"
	"velocity/internal/domain/trade"
	"velocity/internal/engine/command"
	"velocity/internal/engine/matcher"
	"velocity/internal/engine/orderbook"
	"velocity/internal/engine/snapshot"
	"velocity/internal/engine/stopbook"
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

	commandQueue chan command.Command
	tradeQueue   chan *trade.Trade

	lastTradePrice atomic.Int64
	sequence       atomic.Uint64

	done chan struct{} // new
}

func (e *Engine) start() {
	go func() {
		defer close(e.done)
		for cmd := range e.commandQueue {
			switch c := cmd.(type) {
			case command.SubmitOrderCommand:

				if c.Order.Type == constants.StopMarketOrder ||
					c.Order.Type == constants.StopLimitOrder {

					e.stopBook.Add(c.Order)
					c.Order.Status = constants.OrderStatusPending

					e.incrementSequence()

					continue
				}

				trades, err := e.matcher.Match(c.Order)
				if err != nil {
					continue
				}

				e.incrementSequence()

				for _, t := range trades {
					e.lastTradePrice.Store(t.Price)
					metrics.TradesExecuted.Inc()

					e.tradeQueue <- t
				}
				e.processTriggeredStops()

			case command.CancelOrderCommand:

				err := e.stopBook.CancelOrder(c.OrderID)

				if err == nil {
					e.incrementSequence()
					c.Result <- nil
					continue
				}

				err = e.book.CancelOrder(c.OrderID)

				if err == nil {
					e.incrementSequence()
				}

				c.Result <- err

				// in engine.go's start()
			case command.ModifyOrderCommand:
				err := e.book.ModifyOrder(
					c.OrderID,
					c.NewPrice,
					c.NewQuantity,
				)

				if err == nil {
					e.incrementSequence()
				}

				c.Result <- err
			}
		}
	}()
}

func New(symbol string) *Engine {
	book := orderbook.New(symbol)

	e := &Engine{
		symbol:       symbol,
		book:         book,
		matcher:      matcher.New(book),
		stopBook:     stopbook.New(),
		commandQueue: make(chan command.Command, 100000),
		tradeQueue:   make(chan *trade.Trade, 100000),
		// lastTradePrice: 0,
		done: make(chan struct{}),
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

	e.commandQueue <- command.SubmitOrderCommand{
		Order: order,
	}

	return nil
}

// read-only channel accessor.
func (e *Engine) Trades() <-chan *trade.Trade {
	return e.tradeQueue
}

func (e *Engine) OrderBook() *orderbook.OrderBook {
	return e.book
}

func (e *Engine) CancelOrder(orderID string) error {
	resultCh := make(chan error, 1)
	e.commandQueue <- command.CancelOrderCommand{
		OrderID: orderID,
		Result:  resultCh,
	}
	return <-resultCh // blocks until the background goroutine actually processes it
}

func (e *Engine) ModifyOrder(
	orderID string,
	newPrice int64,
	newQuantity int64,
) error {

	resultCh := make(chan error, 1)

	e.commandQueue <- command.ModifyOrderCommand{
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
	close(e.commandQueue) // range loop in start()'s goroutine exits once the channel is closed and drained
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
	return &snapshot.Snapshot{
		Symbol:         e.symbol,
		Sequence:       e.Sequence(),
		LastTradePrice: e.LastTradePrice(),
		ActiveOrders:   e.book.ActiveOrders(),
		StopOrders:     e.stopBook.Orders(),
		CreatedAt:      timeutil.UTCNow(),
	}
}

func (e *Engine) RestoreSnapshot(
	s *snapshot.Snapshot,
) {

	e.sequence.Store(
		s.Sequence,
	)

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