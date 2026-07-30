package marketdata

import (
	"fmt"
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
	trade *trade.Trade,
	book *orderbook.OrderBook,
) {

	fmt.Println("BROADCASTER: DispatchTrade")

	// Publish executed trade
	d.publisher.PublishTrade(trade)
	fmt.Println("Trade published")

	// Publish updated ticker
	d.publisher.PublishTicker(
		trade.Symbol,
		trade.Price,
		book,
	)
    fmt.Println("Ticker published")

	// Publish updated orderbook
	d.publisher.PublishDepth(
		trade.Symbol,
		book,
	)
    fmt.Println("Depth published")

	if candle, ok := d.candleService.Latest(
		trade.Symbol,
		candles.Interval1m,
	); ok {

		_ = d.publisher.PublishKline(
			trade.Symbol,
			candle,
		)
        fmt.Println("Kline published")
	}
}
