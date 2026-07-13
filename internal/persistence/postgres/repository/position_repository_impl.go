package repository

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

	"github.com/google/uuid"
)

type positionRepository struct {
	queries *generated.Queries
}

func NewPositionRepository(db generated.DBTX) PositionRepository {
	return &positionRepository{
		queries: generated.New(db),
	}
}

func (r *positionRepository) Upsert(
	ctx context.Context,
	params generated.UpsertPositionParams,
) error {
	return r.queries.UpsertPosition(ctx, params)
}

func (r *positionRepository) Get(
	ctx context.Context,
	userID uuid.UUID,
	symbol string,
) (generated.Position, error) {
	return r.queries.GetPosition(
		ctx,
		generated.GetPositionParams{
			UserID: userID,
			Symbol: symbol,
		},
	)
}

func (r *positionRepository) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
) ([]generated.Position, error) {
	return r.queries.ListPositionsByUser(ctx, userID)
}