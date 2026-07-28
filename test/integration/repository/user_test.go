package repository

import (
	"testing"
	"time"

	"velocity/internal/persistence/postgres/generated"
	"velocity/test/integration"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateAndGetUser(t *testing.T) {
	tc := integration.NewTestContext(t)

	userID := uuid.New()
	email := uuid.New().String() + "@velocity.dev"

	user, err := tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:           userID,
			Email:        email,
			PasswordHash: "hashed-password",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	)

	require.NoError(t, err)

	require.Equal(t, userID, user.ID)
	require.Equal(t, email, user.Email)
	require.Equal(t, "hashed-password", user.PasswordHash)

	// Fetch by ID
	dbUser, err := tc.UserRepo.GetByID(
		tc.Ctx,
		userID,
	)

	require.NoError(t, err)

	require.Equal(t, user.ID, dbUser.ID)
	require.Equal(t, user.Email, dbUser.Email)
	require.Equal(t, user.PasswordHash, dbUser.PasswordHash)

	// Fetch by Email
	emailUser, err := tc.UserRepo.GetByEmail(
		tc.Ctx,
		email,
	)

	require.NoError(t, err)

	require.Equal(t, user.ID, emailUser.ID)
	require.Equal(t, user.Email, emailUser.Email)
	require.Equal(t, user.PasswordHash, emailUser.PasswordHash)
}