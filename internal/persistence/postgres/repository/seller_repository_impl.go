package repository

import (
	"context"

	"velocity/internal/persistence/postgres/generated"
)

type sellerRepository struct {
	queries *generated.Queries
}

func NewSellerRepository(db generated.DBTX) SellerRepository {
	return &sellerRepository{
		queries: generated.New(db),
	}
}

func (r *sellerRepository) CreateSellerProduct(
	ctx context.Context,
	params generated.CreateSellerProductParams,
) (generated.SellerProduct, error) {
	return r.queries.CreateSellerProduct(ctx, params)
}

func (r *sellerRepository) ListSellerProducts(
	ctx context.Context,
	sellerID int64,
) ([]generated.SellerProduct, error) {
	return r.queries.ListSellerProducts(ctx, sellerID)
}

func (r *sellerRepository) GetAllSellerProducts(
	ctx context.Context,
) ([]generated.SellerProduct, error) {
	return r.queries.GetAllSellerProducts(ctx)
}

func (r *sellerRepository) GetSellerStats(
	ctx context.Context,
	sellerID int64,
) (generated.GetSellerStatsRow, error) {
	return r.queries.GetSellerStats(ctx, sellerID)
}

func (r *sellerRepository) GetSellerActivity(
	ctx context.Context,
	sellerID int64,
) ([]generated.GetSellerActivityRow, error) {
	return r.queries.GetSellerActivity(ctx, sellerID)
}
