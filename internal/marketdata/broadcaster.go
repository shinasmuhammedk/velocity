package marketdata

import (
	"velocity/internal/domain/trade"

	"velocity/internal/engine/orderbook"
)

type Broadcaster struct {
	publisher *Publisher
}

func NewBroadcaster(
	publisher *Publisher,
) *Broadcaster {
	return &Broadcaster{
		publisher: publisher,
	}
}

func (d *Broadcaster) DispatchTrade(
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