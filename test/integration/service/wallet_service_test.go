package integration

import (
	"testing"
	"time"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/service/walletservice"
	"velocity/pkg/errors"
	"velocity/test/integration"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createWalletForServiceTest(t *testing.T, tc *integration.TestContext) (int64, string) {
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

	asset := "USDT"

	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID:    userID,
			Asset:     asset,
			Available: 100000,
			Locked:    0,
		},
	)
	require.NoError(t, err)

	return userID, asset
}

func TestWalletServiceDeposit(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := walletservice.New(tc.WalletRepo)

	userID, asset := createWalletForServiceTest(t, tc)

	err := service.Deposit(
		tc.Ctx,
		userID,
		asset,
		50000,
	)

	require.NoError(t, err)

	wallet, err := service.Get(
		tc.Ctx,
		userID,
		asset,
	)

	require.NoError(t, err)

	require.EqualValues(t, 150000, wallet.Available)
	require.EqualValues(t, 0, wallet.Locked)
}

func TestWalletServiceWithdraw(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := walletservice.New(tc.WalletRepo)

	userID, asset := createWalletForServiceTest(t, tc)

	err := service.Withdraw(tc.Ctx, userID, asset, 25000)
	require.NoError(t, err)

	wallet, err := service.Get(tc.Ctx, userID, asset)
	require.NoError(t, err)

	require.EqualValues(t, 75000, wallet.Available)
	require.EqualValues(t, 0, wallet.Locked)
}

func TestWalletServiceLockFunds(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := walletservice.New(tc.WalletRepo)

	userID, asset := createWalletForServiceTest(t, tc)

	err := service.LockFunds(tc.Ctx, userID, asset, 30000)
	require.NoError(t, err)

	wallet, err := service.Get(tc.Ctx, userID, asset)
	require.NoError(t, err)

	require.EqualValues(t, 70000, wallet.Available)
	require.EqualValues(t, 30000, wallet.Locked)
}

func TestWalletServiceUnlockFunds(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := walletservice.New(tc.WalletRepo)

	userID, asset := createWalletForServiceTest(t, tc)

	require.NoError(t,
		service.LockFunds(tc.Ctx, userID, asset, 30000),
	)

	require.NoError(t,
		service.UnlockFunds(tc.Ctx, userID, asset, 10000),
	)

	wallet, err := service.Get(tc.Ctx, userID, asset)
	require.NoError(t, err)

	require.EqualValues(t, 80000, wallet.Available)
	require.EqualValues(t, 20000, wallet.Locked)
}

func TestWalletServiceConsumeLockedFunds(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := walletservice.New(tc.WalletRepo)

	userID, asset := createWalletForServiceTest(t, tc)

	require.NoError(t,
		service.LockFunds(tc.Ctx, userID, asset, 40000),
	)

	require.NoError(t,
		service.ConsumeLockedFunds(tc.Ctx, userID, asset, 15000),
	)

	wallet, err := service.Get(tc.Ctx, userID, asset)
	require.NoError(t, err)

	require.EqualValues(t, 60000, wallet.Available)
	require.EqualValues(t, 25000, wallet.Locked)
}

func TestWalletServiceWithdrawInsufficientBalance(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := walletservice.New(tc.WalletRepo)

	userID, asset := createWalletForServiceTest(t, tc)

	err := service.Withdraw(
		tc.Ctx,
		userID,
		asset,
		200000,
	)

	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
}

func TestWalletServiceLockInsufficientBalance(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := walletservice.New(tc.WalletRepo)

	userID, asset := createWalletForServiceTest(t, tc)

	err := service.LockFunds(
		tc.Ctx,
		userID,
		asset,
		200000,
	)

	require.ErrorIs(t, err, errors.ErrInsufficientBalance)
}

func TestWalletServiceConsumeInsufficientLockedBalance(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := walletservice.New(tc.WalletRepo)

	userID, asset := createWalletForServiceTest(t, tc)

	require.NoError(t,
		service.LockFunds(tc.Ctx, userID, asset, 10000),
	)

	err := service.ConsumeLockedFunds(
		tc.Ctx,
		userID,
		asset,
		20000,
	)

	require.ErrorIs(t, err, errors.ErrInsufficientLockedBalance)
}

func TestWalletServiceInvalidAmount(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := walletservice.New(tc.WalletRepo)

	userID, asset := createWalletForServiceTest(t, tc)

	err := service.Deposit(
		tc.Ctx,
		userID,
		asset,
		0,
	)

	require.ErrorIs(t, err, errors.ErrInvalidQuantity)
}