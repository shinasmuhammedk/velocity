package worker

import (
	"context"

	"velocity/internal/domain/trade"
	"velocity/internal/marketdata"
)

type TradeConsumer struct {
	worker       TradePersistenceWorker
	dispatcher   *marketdata.Dispatcher
	orderBookFor OrderBookProvider
}

func NewTradeConsumer(
	worker TradePersistenceWorker,
	dispatcher *marketdata.Dispatcher,
	provider OrderBookProvider,
) *TradeConsumer {

	return &TradeConsumer{
		worker:       worker,
		dispatcher:   dispatcher,
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
				book := c.orderBookFor(t.Symbol)

				if book != nil {
					c.dispatcher.DispatchTrade(
						t,
						book,
					)
				}
			}
		}
	}()
}
