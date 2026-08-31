package repository

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

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

func (r *tradeRepository) CreateIfNotExists(
	ctx context.Context,
	params generated.CreateTradeIfNotExistsParams,
) (generated.Trade, error) {
	return r.queries.CreateTradeIfNotExists(ctx, params)
}

func (r *tradeRepository) GetByID(
	ctx context.Context,
	id int64,
) (generated.Trade, error) {
	return r.queries.GetTradeByID(ctx, id)
}

func (r *tradeRepository) ListByUser(
	ctx context.Context,
	userID int64,
) ([]generated.Trade, error) {
	return r.queries.ListTradesByUser(ctx, userID)
}

func (r *tradeRepository) ListBySymbol(
	ctx context.Context,
	symbol string,
) ([]generated.Trade, error) {
	return r.queries.ListTradesBySymbol(ctx, symbol)
}

func (r *tradeRepository) ListBySymbolAsc(
	ctx context.Context,
	symbol string,
) ([]generated.Trade, error) {
	return r.queries.ListTradesBySymbolAsc(
		ctx,
		symbol,
	)
}

func (r *tradeRepository) WithTx(tx pgx.Tx) TradeRepository {
	return &tradeRepository{
		queries: generated.New(tx),
	}
}

func (r *tradeRepository) TradeExists(
	ctx context.Context,
	id int64,
) (bool, error) {
	return r.queries.TradeExists(ctx, id)
}
