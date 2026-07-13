package worker

import (
	"context"
	"velocity/internal/domain/trade"
)

type TradeConsumer struct {
	worker TradePersistenceWorker
}

func NewTradeConsumer(
	worker TradePersistenceWorker,
) *TradeConsumer {
	return &TradeConsumer{
		worker: worker,
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

				err := c.worker.ProcessTrade(ctx, t)
				if err != nil {
					// retry / dlq / logging later
					continue
				}
			}
		}
	}()
}