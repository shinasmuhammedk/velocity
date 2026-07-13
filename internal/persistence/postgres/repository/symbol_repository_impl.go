package repository

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

)

type symbolRepository struct {
	queries *generated.Queries
}

func NewSymbolRepository(db generated.DBTX) SymbolRepository {
	return &symbolRepository{
		queries: generated.New(db),
	}
}

func (r *symbolRepository) Create(
	ctx context.Context,
	params generated.CreateSymbolParams,
) (generated.Symbol, error) {
	return r.queries.CreateSymbol(ctx, params)
}

func (r *symbolRepository) Get(
	ctx context.Context,
	symbol string,
) (generated.Symbol, error) {
	return r.queries.GetSymbol(ctx, symbol)
}

func (r *symbolRepository) List(
	ctx context.Context,
) ([]generated.Symbol, error) {
	return r.queries.ListSymbols(ctx)
}

func (r *symbolRepository) ListActive(
	ctx context.Context,
) ([]generated.Symbol, error) {
	return r.queries.ListActiveSymbols(ctx)
}