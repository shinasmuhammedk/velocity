package repository

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type tradeRepository struct {
	queries *generated.Queries
}

func NewTradeRepository(db generated.DBTX) TradeRepository {
	return &tradeRepository{
		queries: generated.New(db),
	}
}

func (r *tradeRepository) Create(
	ctx context.Context,
	params generated.CreateTradeParams,
) (generated.Trade, error) {
	return r.queries.CreateTrade(ctx, params)
}

func (r *tradeRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (generated.Trade, error) {
	return r.queries.GetTradeByID(ctx, id)
}

func (r *tradeRepository) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
) ([]generated.Trade, error) {
	return r.queries.ListTradesByUser(ctx, userID)
}

func (r *tradeRepository) ListBySymbol(
	ctx context.Context,
	symbol string,
) ([]generated.Trade, error) {
	return r.queries.ListTradesBySymbol(ctx, symbol)
}

func (r *tradeRepository) WithTx(tx pgx.Tx) TradeRepository {
	return &tradeRepository{
		queries: generated.New(tx),
	}
}

func (r *tradeRepository) TradeExists(
    ctx context.Context,
    id uuid.UUID,
) (bool, error) {
    return r.queries.TradeExists(ctx, id)
}