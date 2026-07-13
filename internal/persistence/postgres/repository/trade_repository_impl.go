package repository

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

	"github.com/google/uuid"
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