package integration

import (
	"testing"
	"time"

	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/postgres/repository"
	"velocity/internal/service/walletservice"
	"velocity/test/integration"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// newWalletTestService builds a walletservice.Service against the real
// DB, plus a WalletTransactionRepository (not currently exposed on
// TestContext) needed to seed/inspect ledger rows directly.
func newWalletTestService(tc *integration.TestContext) (*walletservice.Service, repository.WalletTransactionRepository) {
	txRepo := repository.NewWalletTransactionRepository(tc.DB)
	svc := walletservice.New(tc.WalletRepo, txRepo)
	return svc, txRepo
}

func createWalletTestUser(t *testing.T, tc *integration.TestContext) int64 {
	t.Helper()

	userID := time.Now().UnixNano()

	_, err := tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:        userID,
			Email:     uuid.NewString() + "@test.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	return userID
}

func createWalletTestWallet(
	t *testing.T,
	tc *integration.TestContext,
	userID int64,
	asset string,
) {
	t.Helper()

	_, err := tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID:    userID,
			Asset:     asset,
			Available: 0,
			Locked:    0,
		},
	)
	require.NoError(t, err)
}

// TestListTransactionsByAsset_ReturnsRecordedTransactions verifies
// that deposits and withdrawals appear in the wallet ledger in
// reverse chronological order.
func TestListTransactionsByAsset_ReturnsRecordedTransactions(t *testing.T) {
	tc := integration.NewTestContext(t)
	svc, _ := newWalletTestService(tc)

	userID := createWalletTestUser(t, tc)
	asset := "USD_" + uuid.NewString()[:8]
	createWalletTestWallet(t, tc, userID, asset)

	require.NoError(t, svc.Deposit(tc.Ctx, userID, asset, 500))
	require.NoError(t, svc.Withdraw(tc.Ctx, userID, asset, 200))

	txs, err := svc.ListTransactionsByAsset(tc.Ctx, userID, asset)
	require.NoError(t, err)
	require.Len(t, txs, 2)

	// ORDER BY created_at DESC, id DESC - most recent (the withdrawal)
	// first.
	require.Equal(t, "WITHDRAWAL", txs[0].Type)
	require.Equal(t, int64(200), txs[0].Amount)

	require.Equal(t, "DEPOSIT", txs[1].Type)
	require.Equal(t, int64(500), txs[1].Amount)

	wallet, err := tc.WalletRepo.Get(tc.Ctx, userID, asset)
	require.NoError(t, err)
	require.Equal(t, int64(300), wallet.Available)
}

// TestListTransactionsByAsset_ScopedByAsset ensures a transaction on
// one asset never leaks into the ledger of another asset for the same
// user.
func TestListTransactionsByAsset_ScopedByAsset(t *testing.T) {
	tc := integration.NewTestContext(t)
	svc, _ := newWalletTestService(tc)

	userID := createWalletTestUser(t, tc)

	assetA := "AAA_" + uuid.NewString()[:8]
	assetB := "BBB_" + uuid.NewString()[:8]

	createWalletTestWallet(t, tc, userID, assetA)
	createWalletTestWallet(t, tc, userID, assetB)

	require.NoError(t, svc.Deposit(tc.Ctx, userID, assetA, 100))
	require.NoError(t, svc.Deposit(tc.Ctx, userID, assetB, 999))

	txsA, err := svc.ListTransactionsByAsset(tc.Ctx, userID, assetA)
	require.NoError(t, err)
	require.Len(t, txsA, 1)
	require.Equal(t, int64(100), txsA[0].Amount)
	require.Equal(t, assetA, txsA[0].Asset)
}

// TestListTransactionsByAsset_ScopedByUser ensures one user can never
// see another user's ledger entries, even for the same asset.
func TestListTransactionsByAsset_ScopedByUser(t *testing.T) {
	tc := integration.NewTestContext(t)
	svc, _ := newWalletTestService(tc)

	userA := createWalletTestUser(t, tc)
	userB := createWalletTestUser(t, tc)
	asset := "SHARED_" + uuid.NewString()[:8]

	createWalletTestWallet(t, tc, userA, asset)
	createWalletTestWallet(t, tc, userB, asset)

	require.NoError(t, svc.Deposit(tc.Ctx, userA, asset, 111))
	require.NoError(t, svc.Deposit(tc.Ctx, userB, asset, 222))

	txsA, err := svc.ListTransactionsByAsset(tc.Ctx, userA, asset)
	require.NoError(t, err)
	require.Len(t, txsA, 1)
	require.Equal(t, int64(111), txsA[0].Amount)
}

// TestListTransactionsByAsset_NoTransactions confirms a wallet with no
// activity returns an empty (not nil-panicking, not erroring) list.
func TestListTransactionsByAsset_NoTransactions(t *testing.T) {
	tc := integration.NewTestContext(t)
	svc, _ := newWalletTestService(tc)

	userID := createWalletTestUser(t, tc)
	asset := "EMPTY_" + uuid.NewString()[:8]
	createWalletTestWallet(t, tc, userID, asset)

	txs, err := svc.ListTransactionsByAsset(tc.Ctx, userID, asset)
	require.NoError(t, err)
	require.Empty(t, txs)
}

// TestDeposit_ViaHandlerPath_AppearsInLedger ensures that the Deposit()
// method used by the wallet HTTP handler updates the wallet balance and
// records the corresponding transaction in the wallet ledger.
func TestDeposit_ViaHandlerPath_AppearsInLedger(t *testing.T) {
	tc := integration.NewTestContext(t)
	svc, _ := newWalletTestService(tc)

	userID := createWalletTestUser(t, tc)
	asset := "GAP_" + uuid.NewString()[:8]
	createWalletTestWallet(t, tc, userID, asset)

	require.NoError(t, svc.Deposit(tc.Ctx, userID, asset, 500))

	wallet, err := tc.WalletRepo.Get(tc.Ctx, userID, asset)
	require.NoError(t, err)
	require.Equal(t, int64(500), wallet.Available)

	txs, err := svc.ListTransactionsByAsset(tc.Ctx, userID, asset)
	require.NoError(t, err)
	require.Len(t, txs, 1)
	require.Equal(t, "DEPOSIT", txs[0].Type)
	require.Equal(t, int64(500), txs[0].Amount)
	require.Equal(t, asset, txs[0].Asset)
	require.Equal(t, userID, txs[0].UserID)
}

// TestWithdraw_ViaHandlerPath_AppearsInLedger ensures that the Withdraw()
// method used by the wallet HTTP handler updates the wallet balance and
// records the corresponding transaction in the wallet ledger.
func TestWithdraw_ViaHandlerPath_AppearsInLedger(t *testing.T) {
	tc := integration.NewTestContext(t)
	svc, _ := newWalletTestService(tc)

	userID := createWalletTestUser(t, tc)
	asset := "WITHDRAW_" + uuid.NewString()[:8]
	createWalletTestWallet(t, tc, userID, asset)

	require.NoError(t, svc.Deposit(tc.Ctx, userID, asset, 500))
	require.NoError(t, svc.Withdraw(tc.Ctx, userID, asset, 200))

	wallet, err := tc.WalletRepo.Get(tc.Ctx, userID, asset)
	require.NoError(t, err)
	require.Equal(t, int64(300), wallet.Available)

	txs, err := svc.ListTransactionsByAsset(tc.Ctx, userID, asset)
	require.NoError(t, err)

	require.Len(t, txs, 2)

	require.Equal(t, "WITHDRAWAL", txs[0].Type)
	require.Equal(t, int64(200), txs[0].Amount)
	require.Equal(t, asset, txs[0].Asset)
	require.Equal(t, userID, txs[0].UserID)

	require.Equal(t, "DEPOSIT", txs[1].Type)
	require.Equal(t, int64(500), txs[1].Amount)
}
