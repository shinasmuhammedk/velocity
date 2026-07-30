package seed

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SeedUsers(
	ctx context.Context,
	db *pgxpool.Pool,
) error {

	q := generated.New(db)

	for _, u := range Users {

		_, err := q.CreateUser(
			ctx,
			generated.CreateUserParams{
				ID: uuid.MustParse(u.ID),

				Email: u.Email,

				PasswordHash: u.Password,
			},
		)

		if err != nil {
			return err
		}
	}

	return nil
}