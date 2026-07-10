package engine

import (
	"velocity/internal/domain/order"
	"velocity/internal/domain/trade"
	"velocity/internal/engine/matcher"
	"velocity/internal/engine/orderbook"
)

type Engine struct {
	book    *orderbook.OrderBook
	matcher *matcher.Matcher

	orderQueue chan *order.Order
	tradeQueue chan *trade.Trade
}

func (e *Engine) start() {
	go func() {
		for order := range e.orderQueue {

			trades, err := e.matcher.Match(order)
			if err != nil {
				continue
			}

			for _, t := range trades {
				e.tradeQueue <- t
			}
		}
	}()
}

func New(symbol string) *Engine {
	book := orderbook.New(symbol)

	e := &Engine{
		book:       book,
		matcher:    matcher.New(book),
		orderQueue: make(chan *order.Order, 100000),
		tradeQueue: make(chan *trade.Trade, 100000),
	}

	e.start()

	return e
}

func (e *Engine) SubmitOrder(order *order.Order) error {
	e.orderQueue <- order
	return nil
}

//read-only channel accessor.
func (e *Engine) Trades() <-chan *trade.Trade {
	return e.tradeQueue
}


func (e *Engine) OrderBook() *orderbook.OrderBook {
	return e.book
}
