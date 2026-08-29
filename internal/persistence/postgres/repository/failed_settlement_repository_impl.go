package repository

import (
	"context"

	"github.com/google/uuid"

	"velocity/internal/persistence/postgres/generated"
)

type failedSettlementRepository struct {
	queries *generated.Queries
}

func NewFailedSettlementRepository(db generated.DBTX) FailedSettlementRepository {
	return &failedSettlementRepository{
		queries: generated.New(db),
	}
}

func (r *failedSettlementRepository) Create(
	ctx context.Context,
	params generated.CreateFailedSettlementParams,
) (generated.FailedSettlement, error) {
	return r.queries.CreateFailedSettlement(ctx, params)
}

func (r *failedSettlementRepository) Get(
	ctx context.Context,
	id uuid.UUID,
) (generated.FailedSettlement, error) {
	return r.queries.GetFailedSettlement(ctx, id)
}

func (r *failedSettlementRepository) ListUnresolved(
	ctx context.Context,
) ([]generated.FailedSettlement, error) {
	return r.queries.ListUnresolvedFailedSettlements(ctx)
}

func (r *failedSettlementRepository) IncrementRetryCount(
	ctx context.Context,
	id uuid.UUID,
) error {
	return r.queries.IncrementFailedSettlementRetryCount(ctx, id)
}

func (r *failedSettlementRepository) Resolve(
	ctx context.Context,
	id uuid.UUID,
) error {
	return r.queries.ResolveFailedSettlement(ctx, id)
}

func (r *failedSettlementRepository) MarkDead(
	ctx context.Context,
	id uuid.UUID,
) error {
	return r.queries.MarkFailedSettlementDead(ctx, id)
}
