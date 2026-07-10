package repository

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

	"github.com/jackc/pgx/v5/pgtype"
)

type UserRepository interface {
	Create(
		ctx context.Context,
		params generated.CreateUserParams,
	) (*generated.User, error)

	GetByID(
		ctx context.Context,
		id pgtype.UUID,
	) (*generated.User, error)

	GetByEmail(
		ctx context.Context,
		email string,
	) (*generated.User, error)

	Delete(
		ctx context.Context,
		id pgtype.UUID,
	) error
}