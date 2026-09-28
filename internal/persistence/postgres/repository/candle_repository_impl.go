package repository

import (
	"context"
	"time"

	"velocity/internal/persistence/postgres/generated"
)

type candleRepository struct {
	queries *generated.Queries
}

func NewCandleRepository(db generated.DBTX) CandleRepository {
	return &candleRepository{
		queries: generated.New(db),
	}
}

func (r *candleRepository) Upsert(
	ctx context.Context,
	params generated.UpsertCandleParams,
) (generated.Candle, error) {
	return r.queries.UpsertCandle(ctx, params)
}

func (r *candleRepository) ListBySymbolInterval(
	ctx context.Context,
	symbol string,
	interval string,
	limit int32,
) ([]generated.Candle, error) {
	return r.queries.ListCandlesBySymbolInterval(
		ctx,
		generated.ListCandlesBySymbolIntervalParams{
			Symbol:   symbol,
			Interval: interval,
			Limit:    limit,
		},
	)
}

func (r *candleRepository) ListBySymbolIntervalRange(
	ctx context.Context,
	symbol string,
	interval string,
	start time.Time,
	end time.Time,
	limit int32,
) ([]generated.Candle, error) {
	return r.queries.ListCandlesBySymbolIntervalRange(
		ctx,
		generated.ListCandlesBySymbolIntervalRangeParams{
			Symbol:     symbol,
			Interval:   interval,
			OpenTime:   start,
			OpenTime_2: end,
			Limit:      limit,
		},
	)
}
