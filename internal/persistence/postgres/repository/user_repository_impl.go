package repository

import (
	"context"

	"velocity/internal/persistence/postgres/generated"

	"github.com/google/uuid"
)

type userRepository struct {
	queries *generated.Queries
}

func NewUserRepository(db generated.DBTX) UserRepository {
	return &userRepository{
		queries: generated.New(db),
	}
}

func (r *userRepository) Create(ctx context.Context, params generated.CreateUserParams) (generated.User, error) {
	return r.queries.CreateUser(ctx, params)
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (generated.User, error) {
	return r.queries.GetUserByID(ctx, id)
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (generated.User, error) {
	return r.queries.GetUserByEmail(ctx, email)
}
