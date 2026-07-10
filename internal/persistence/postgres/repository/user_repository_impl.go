package repository

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepository struct {
	queries *generated.Queries
}

func NewUserRepository(
	db *pgxpool.Pool,
) UserRepository {
	return &userRepository{
		queries: generated.New(db),
	}
}

func (r *userRepository) Create(
	ctx context.Context,
	params generated.CreateUserParams,
) (*generated.User, error) {

	user, err := r.queries.CreateUser(
		ctx,
		params,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetByID(
	ctx context.Context,
	id pgtype.UUID,
) (*generated.User, error) {

	user, err := r.queries.GetUserByID(
		ctx,
		id,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetByEmail(
	ctx context.Context,
	email string,
) (*generated.User, error) {

	user, err := r.queries.GetUserByEmail(
		ctx,
		email,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) Delete(
	ctx context.Context,
	id pgtype.UUID,
) error {

	return r.queries.DeleteUser(
		ctx,
		id,
	)
}