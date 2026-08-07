package repository

import (
	"testing"
	"time"

	"velocity/internal/persistence/postgres/generated"
	"velocity/test/integration"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPositionRepository(t *testing.T) {
	tc := integration.NewTestContext(t)

	// -----------------------------
	// Create User
	// -----------------------------
	userID := time.Now().UnixNano()

	_, err := tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:           userID,
			Email:        uuid.New().String() + "@test.com",
			// PasswordHash: "password",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	)
	require.NoError(t, err)

	// -----------------------------
	// Create Symbol
	// -----------------------------
	symbol := "BTCUSDT_" + uuid.New().String()[:8]

	_, err = tc.SymbolRepo.Create(
		tc.Ctx,
		generated.CreateSymbolParams{
			Symbol:      symbol,
			DisplayName: "Bitcoin/Tether",
			TickSize:    1,
			LotSize:     1,
			IsActive:    true,
		},
	)
	require.NoError(t, err)

	// -----------------------------
	// First Upsert (INSERT)
	// -----------------------------
	err = tc.PositionRepo.Upsert(
		tc.Ctx,
		generated.UpsertPositionParams{
			UserID:   userID,
			Symbol:   symbol,
			Quantity: 10,
		},
	)
	require.NoError(t, err)

	position, err := tc.PositionRepo.Get(
		tc.Ctx,
		userID,
		symbol,
	)

	require.NoError(t, err)
	require.Equal(t, userID, position.UserID)
	require.Equal(t, symbol, position.Symbol)
	require.EqualValues(t, 10, position.Quantity)

	// -----------------------------
	// Second Upsert (UPDATE)
	// Should add quantity
	// -----------------------------
	err = tc.PositionRepo.Upsert(
		tc.Ctx,
		generated.UpsertPositionParams{
			UserID:   userID,
			Symbol:   symbol,
			Quantity: 5,
		},
	)
	require.NoError(t, err)

	position, err = tc.PositionRepo.Get(
		tc.Ctx,
		userID,
		symbol,
	)

	require.NoError(t, err)
	require.EqualValues(t, 15, position.Quantity)

	// -----------------------------
	// ListByUser
	// -----------------------------
	positions, err := tc.PositionRepo.ListByUser(
		tc.Ctx,
		userID,
	)

	require.NoError(t, err)
	require.Len(t, positions, 1)

	require.Equal(t, userID, positions[0].UserID)
	require.Equal(t, symbol, positions[0].Symbol)
	require.EqualValues(t, 15, positions[0].Quantity)
}