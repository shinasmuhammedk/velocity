package seed

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

	"github.com/jackc/pgx/v5/pgxpool"
)

func SeedPositions(
	ctx context.Context,
	db *pgxpool.Pool,
) error {

	q := generated.New(db)

	for _, p := range Positions {

		err := q.UpsertPosition(
			ctx,
			generated.UpsertPositionParams{
				UserID: p.UserID,
				Symbol: p.Symbol,

				Quantity: p.Quantity,
			},
		)

		if err != nil {
			return err
		}
	}

	return nil
}