package candles

import (
	"context"

	"go.uber.org/zap"

	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
)

// CandlePersister durably writes closed candles to Postgres.
//
// It runs entirely off the matching hot path: it owns its own
// goroutine and only ever reads from Manager.Closed(), which the
// matching engine's goroutine sends to non-blockingly (see
// Manager/updateInterval). Nothing here can make an order-matching
// call wait on a database round trip.
//
// Writes are idempotent upserts keyed on (symbol, interval,
// open_time), so redundant persistence of an already-durable candle
// (e.g. one replayed during startup backfill) is harmless.
type CandlePersister struct {
	manager *Manager
	repo    repository.CandleRepository
	logger  *zap.Logger
}

func NewCandlePersister(
	manager *Manager,
	repo repository.CandleRepository,
	logger *zap.Logger,
) *CandlePersister {
	return &CandlePersister{
		manager: manager,
		repo:    repo,
		logger:  logger,
	}
}

// Start begins draining closed candles until ctx is cancelled or
// Manager's closed channel is closed. It runs in its own goroutine
// and returns immediately.
func (p *CandlePersister) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case candle, ok := <-p.manager.Closed():
				if !ok {
					return
				}

				p.persist(ctx, candle)
			}
		}
	}()
}

func (p *CandlePersister) persist(ctx context.Context, candle *Candle) {
	_, err := p.repo.Upsert(ctx, generated.UpsertCandleParams{
		Symbol:      candle.Symbol,
		Interval:    string(candle.Interval),
		OpenTime:    candle.OpenTime,
		CloseTime:   candle.CloseTime,
		Open:        candle.Open,
		High:        candle.High,
		Low:         candle.Low,
		Close:       candle.Close,
		Volume:      candle.Volume,
		QuoteVolume: candle.QuoteVolume,
		TradeCount:  int64(candle.TradeCount),
	})

	if err != nil {
		p.logger.Error(
			"candle persister: failed to persist closed candle",
			zap.String("symbol", candle.Symbol),
			zap.String("interval", string(candle.Interval)),
			zap.Time("open_time", candle.OpenTime),
			zap.Error(err),
		)

		// Deliberately not retried here: the next candle close for
		// this symbol/interval, and more importantly the bounded
		// startup backfill (see backfill.go), will both re-derive
		// and re-upsert this candle from trades, which are the
		// actual source of truth. A dedicated retry queue mirroring
		// failed_settlements would be reasonable follow-up work if
		// write failures turn out to be frequent in practice.
	}
}
