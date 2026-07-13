package worker

import (
	"context"
	"velocity/internal/domain/trade"
)

type TradePersistenceWorker interface {
	ProcessTrade(ctx context.Context, t *trade.Trade)error
}