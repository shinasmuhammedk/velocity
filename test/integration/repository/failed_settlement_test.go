package repository

import (
	"testing"
	"time"

	"velocity/internal/persistence/postgres/generated"
	"velocity/test/integration"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFailedSettlement_RetryAndDeadLetter(t *testing.T) {
	tc := integration.NewTestContext(t)

	base := time.Now().UnixNano()

	failed, err := tc.FailedSettlementRepo.Create(
		tc.Ctx,
		generated.CreateFailedSettlementParams{
			TradeID:      base,
			BuyOrderID:   base + 1,
			SellOrderID:  base + 2,
			BuyerID:      base + 3,
			SellerID:     base + 4,
			Symbol:       "BTCUSDT_" + uuid.New().String()[:8],
			Price:        50000,
			Quantity:     1,
			ErrorMessage: "insufficient locked funds",
		},
	)

	require.NoError(t, err)
	require.Equal(t, int32(0), failed.RetryCount)
	require.False(t, failed.Resolved)
	require.False(t, failed.IsDead)

	// Increment retry count until the configured maximum.
	for i := int32(1); i <= 10; i++ {
		err = tc.FailedSettlementRepo.IncrementRetryCount(
			tc.Ctx,
			failed.ID,
		)

		require.NoError(t, err)

		failed, err = tc.FailedSettlementRepo.Get(
			tc.Ctx,
			failed.ID,
		)

		require.NoError(t, err)
		require.Equal(t, i, failed.RetryCount)
		require.False(t, failed.Resolved)
		require.False(t, failed.IsDead)
	}

	// Move the exhausted settlement into the dead-letter state.
	err = tc.FailedSettlementRepo.MarkDead(
		tc.Ctx,
		failed.ID,
	)

	require.NoError(t, err)

	failed, err = tc.FailedSettlementRepo.Get(
		tc.Ctx,
		failed.ID,
	)

	require.NoError(t, err)
	require.Equal(t, int32(10), failed.RetryCount)
	require.False(t, failed.Resolved)
	require.True(t, failed.IsDead)

	// Dead settlements must not be returned by ListUnresolved.
	unresolved, err := tc.FailedSettlementRepo.ListUnresolved(
		tc.Ctx,
	)

	require.NoError(t, err)

	for _, item := range unresolved {
		require.NotEqual(t, failed.ID, item.ID)
	}
}

func TestFailedSettlement_ResolveAfterRetry(t *testing.T) {
	tc := integration.NewTestContext(t)

	base := time.Now().UnixNano()

	failed, err := tc.FailedSettlementRepo.Create(
		tc.Ctx,
		generated.CreateFailedSettlementParams{
			TradeID:      base,
			BuyOrderID:   base + 1,
			SellOrderID:  base + 2,
			BuyerID:      base + 3,
			SellerID:     base + 4,
			Symbol:       "BTCUSDT_" + uuid.New().String()[:8],
			Price:        50000,
			Quantity:     1,
			ErrorMessage: "temporary settlement failure",
		},
	)

	require.NoError(t, err)

	// Simulate several failed attempts.
	for i := 0; i < 3; i++ {
		err = tc.FailedSettlementRepo.IncrementRetryCount(
			tc.Ctx,
			failed.ID,
		)

		require.NoError(t, err)
	}

	// Settlement eventually succeeds.
	err = tc.FailedSettlementRepo.Resolve(
		tc.Ctx,
		failed.ID,
	)

	require.NoError(t, err)

	failed, err = tc.FailedSettlementRepo.Get(
		tc.Ctx,
		failed.ID,
	)

	require.NoError(t, err)
	require.Equal(t, int32(3), failed.RetryCount)
	require.True(t, failed.Resolved)
	require.False(t, failed.IsDead)
	require.True(t, failed.ResolvedAt.Valid)

	// Resolved settlement must not be returned as unresolved.
	unresolved, err := tc.FailedSettlementRepo.ListUnresolved(
		tc.Ctx,
	)

	require.NoError(t, err)

	for _, item := range unresolved {
		require.NotEqual(t, failed.ID, item.ID)
	}
}
