package repository

import (
	"testing"
	"time"

	"velocity/internal/persistence/postgres/generated"
	"velocity/test/integration"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateAndGetWallet(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create a real user first
	user, err := tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:           uuid.New(),
			Email:        uuid.New().String() + "@velocity.dev",
			PasswordHash: "password123",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	)
	require.NoError(t, err)

	// Create wallet for that user
	wallet, err := tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID:    user.ID,
			Asset:     "USDT",
			Available: 100000,
			Locked:    25000,
		},
	)
	require.NoError(t, err)

	require.Equal(t, user.ID, wallet.UserID)
	require.Equal(t, "USDT", wallet.Asset)
	require.EqualValues(t, 100000, wallet.Available)
	require.EqualValues(t, 25000, wallet.Locked)

	// Fetch wallet
	dbWallet, err := tc.WalletRepo.Get(
		tc.Ctx,
		user.ID,
		"USDT",
	)
	require.NoError(t, err)

	require.Equal(t, wallet.ID, dbWallet.ID)
	require.Equal(t, wallet.UserID, dbWallet.UserID)
	require.Equal(t, wallet.Asset, dbWallet.Asset)
	require.Equal(t, wallet.Available, dbWallet.Available)
	require.Equal(t, wallet.Locked, dbWallet.Locked)

	// Update wallet
	err = tc.WalletRepo.Update(
		tc.Ctx,
		generated.UpdateWalletParams{
			ID:        wallet.ID,
			Available: 75000,
			Locked:    50000,
		},
	)
	require.NoError(t, err)

	updatedWallet, err := tc.WalletRepo.Get(
		tc.Ctx,
		user.ID,
		"USDT",
	)
	require.NoError(t, err)

	require.EqualValues(t, 75000, updatedWallet.Available)
	require.EqualValues(t, 50000, updatedWallet.Locked)
}