package worker

import (
	"context"

	"velocity/internal/domain/trade"
	"velocity/internal/marketdata"
)

type TradeConsumer struct {
	worker       TradePersistenceWorker
	publisher    *marketdata.Publisher
	orderBookFor OrderBookProvider
}

func NewTradeConsumer(
	worker TradePersistenceWorker,
	publisher *marketdata.Publisher,
	provider OrderBookProvider,
) *TradeConsumer {

	return &TradeConsumer{
		worker:       worker,
		publisher:    publisher,
		orderBookFor: provider,
	}
}

func (c *TradeConsumer) Start(
	ctx context.Context,
	trades <-chan *trade.Trade,
) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case t := <-trades:
				if t == nil {
					continue
				}

				// Persist trade
				err := c.worker.ProcessTrade(ctx, t)
				if err != nil {
					// TODO:
					// retry queue
					// dead letter queue
					// structured logging
					continue
				}

				// Publish trade event
				c.publisher.PublishTrade(t)

				// Publish market data updates
				book := c.orderBookFor(t.Symbol)

				if book != nil {
					c.publisher.PublishTicker(
						t.Symbol,
						book,
					)

					c.publisher.PublishDepth(
						t.Symbol,
						book,
					)
				}
			}
		}
	}()
}