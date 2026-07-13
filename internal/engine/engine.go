package engine

import (
	"errors"
	"velocity/internal/domain/order"
	"velocity/internal/domain/trade"
	"velocity/internal/engine/command"
	"velocity/internal/engine/matcher"
	"velocity/internal/engine/orderbook"
	"velocity/internal/engine/stopbook"
	"velocity/pkg/constants"
)

type Engine struct {
	book     *orderbook.OrderBook
	matcher  *matcher.Matcher
	stopBook *stopbook.StopBook

	commandQueue chan command.Command
	tradeQueue   chan *trade.Trade

	lastTradePrice int64
}

func (e *Engine) start() {
	go func() {
		for cmd := range e.commandQueue {
			switch c := cmd.(type) {
			case command.SubmitOrderCommand:

				if c.Order.Type == constants.StopMarketOrder ||
					c.Order.Type == constants.StopLimitOrder {

					e.stopBook.Add(c.Order)
					c.Order.Status = constants.OrderStatusPending
					continue
				}

				trades, err := e.matcher.Match(c.Order)
				if err != nil {
					continue
				}
				for _, t := range trades {
					e.lastTradePrice = t.Price
					e.tradeQueue <- t
				}

				e.processTriggeredStops()

			case command.CancelOrderCommand:

				err := e.stopBook.CancelOrder(c.OrderID)
				if err == nil {
					c.Result <- nil
					continue
				}

				err = e.book.CancelOrder(c.OrderID)
				c.Result <- err

				// in engine.go's start()
			case command.ModifyOrderCommand:
				err := e.book.ModifyOrder(c.OrderID, c.NewPrice, c.NewQuantity)
				c.Result <- err
			}
		}
	}()
}

func New(symbol string) *Engine {
	book := orderbook.New(symbol)

	e := &Engine{
		book:           book,
		matcher:        matcher.New(book),
		stopBook:       stopbook.New(),
		commandQueue:   make(chan command.Command, 100000),
		tradeQueue:     make(chan *trade.Trade, 100000),
		lastTradePrice: 0,
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

		return errors.New(
			"post only orders must be limit orders",
		)
	}

	// STOP order validation
	if order.Type == constants.StopMarketOrder ||
		order.Type == constants.StopLimitOrder {

		if order.StopPrice <= 0 {
			return errors.New("invalid stop price")
		}

		if order.Side == constants.OrderSideBuy &&
			e.lastTradePrice > 0 &&
			order.StopPrice <= e.lastTradePrice {

			return errors.New(
				"buy stop must be above market price",
			)
		}

		if order.Side == constants.OrderSideSell &&
			e.lastTradePrice > 0 &&
			order.StopPrice >= e.lastTradePrice {

			return errors.New(
				"sell stop must be below market price",
			)
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

// Engine.ModifyOrder
func (e *Engine) ModifyOrder(orderID string, newPrice, newQuantity int64) error {
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
		triggered := e.stopBook.Trigger(e.lastTradePrice)

		if len(triggered) == 0 {
			return
		}

		for _, stopOrder := range triggered {

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
				e.lastTradePrice = trade.Price
				e.tradeQueue <- trade
			}
		}
	}
}

func (e *Engine) LastTradePrice() int64 {
	return e.lastTradePrice
}

func (e *Engine) StopBook() *stopbook.StopBook {
	return e.stopBook
}
