package marketdata

import (
	"velocity/internal/domain/trade"

	"velocity/internal/engine/orderbook"
)

type Dispatcher struct {
	publisher *Publisher
}

func NewDispatcher(
	publisher *Publisher,
) *Dispatcher {
	return &Dispatcher{
		publisher: publisher,
	}
}

func (d *Dispatcher) DispatchTrade(
	trade *trade.Trade,
	book *orderbook.OrderBook,
) {

	// Publish executed trade
	d.publisher.PublishTrade(trade)

	// Publish updated ticker
	d.publisher.PublishTicker(
		trade.Symbol,
        trade.Price,
		book,
	)

	// Publish updated orderbook
	d.publisher.PublishDepth(
		trade.Symbol,
		book,
	)
}