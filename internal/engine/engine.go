package engine

import (
	"velocity/internal/domain/order"
	"velocity/internal/domain/trade"
	"velocity/internal/engine/command"
	"velocity/internal/engine/matcher"
	"velocity/internal/engine/orderbook"
)

type Engine struct {
	book    *orderbook.OrderBook
	matcher *matcher.Matcher

	commandQueue chan command.Command
	tradeQueue chan *trade.Trade
}

func (e *Engine) start() {
	go func() {
		for cmd := range e.commandQueue {
			switch c := cmd.(type) {
			case command.SubmitOrderCommand:
				trades, err := e.matcher.Match(c.Order)
				if err != nil {
					continue
				}
				for _, t := range trades {
					e.tradeQueue <- t
				}

			case command.CancelOrderCommand:
				err := e.book.CancelOrder(c.OrderID)
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
		book:         book,
		matcher:      matcher.New(book),
		commandQueue: make(chan command.Command, 100000),
		tradeQueue:   make(chan *trade.Trade, 100000),
	}

	e.start()

	return e
}

func (e *Engine) SubmitOrder(order *order.Order) error {
	e.commandQueue <- command.SubmitOrderCommand{Order: order}
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
