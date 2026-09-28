package marketdata

import (
	"velocity/internal/analytics/candles"
	"velocity/internal/domain/trade"

	"velocity/internal/engine/orderbook"
)

type Broadcaster struct {
	publisher     *Publisher
	candleService *candles.Service
}

func NewBroadcaster(
	publisher *Publisher,
	candleService *candles.Service,
) *Broadcaster {
	return &Broadcaster{
		publisher:     publisher,
		candleService: candleService,
	}
}

func (d *Broadcaster) DispatchTrade(
	trade trade.Trade,
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

	if candle, ok := d.candleService.Latest(
		trade.Symbol,
		candles.Interval1m,
	); ok {

		_ = d.publisher.PublishKline(
			trade.Symbol,
			candle,
		)
	}
}
