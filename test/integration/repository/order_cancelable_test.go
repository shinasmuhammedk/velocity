package repository

import (
	"testing"
	"time"

	"velocity/internal/persistence/postgres/generated"
	"velocity/test/integration"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// TestListCancelableOrders verifies the two queries that back the
// DELETE /orders (cancel-all) endpoint: they must return only orders
// in a cancelable status (OPEN, PARTIALLY_FILLED, PENDING) and must
// correctly scope by symbol when one is provided.
func TestListCancelableOrders(t *testing.T) {

	tc := integration.NewTestContext(t)

	base := time.Now().UnixNano()
	userID := base

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

	symbolA := "AAA_" + uuid.NewString()[:8]
	symbolB := "BBB_" + uuid.NewString()[:8]

	for _, s := range []string{symbolA, symbolB} {
		_, err := tc.SymbolRepo.Create(
			tc.Ctx,
			generated.CreateSymbolParams{
				Symbol:      s,
				DisplayName: s,
				TickSize:    1,
				LotSize:     1,
				IsActive:    true,
			},
		)
		require.NoError(t, err)
	}

	// One order per status/symbol combination we care about.
	type seed struct {
		id     int64
		symbol string
		status string
	}

	seeds := []seed{
		{base + 1, symbolA, "OPEN"},
		{base + 2, symbolA, "PARTIALLY_FILLED"},
		{base + 3, symbolA, "PENDING"},
		{base + 4, symbolA, "FILLED"},    // not cancelable
		{base + 5, symbolA, "CANCELLED"}, // not cancelable
		{base + 6, symbolB, "OPEN"},      // cancelable, different symbol
	}

	for _, s := range seeds {
		_, err := tc.OrderRepo.Create(
			tc.Ctx,
			generated.CreateOrderParams{
				ID:          s.id,
				UserID:      userID,
				Symbol:      s.symbol,
				Side:        "BUY",
				OrderType:   "LIMIT",
				TimeInForce: "GTC",
				Status:      s.status,

				Price: pgtype.Int8{
					Int64: 100,
					Valid: true,
				},

				StopPrice: 0,
				Quantity:  10,
				Remaining: 10,
				Filled:    0,

				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		)
		require.NoError(t, err)
	}

	// ---------- ListCancelableOrdersByUser: across all symbols ----------

	all, err := tc.OrderRepo.ListCancelableOrdersByUser(
		tc.Ctx,
		userID,
	)
	require.NoError(t, err)

	allIDs := make([]int64, len(all))
	for i, o := range all {
		allIDs[i] = o.ID
	}

	require.ElementsMatch(
		t,
		[]int64{base + 1, base + 2, base + 3, base + 6},
		allIDs,
		"only OPEN/PARTIALLY_FILLED/PENDING orders across both symbols should be returned",
	)

	// ---------- ListCancelableOrdersByUserAndSymbol: scoped to symbolA ----------

	scoped, err := tc.OrderRepo.ListCancelableOrdersByUserAndSymbol(
		tc.Ctx,
		generated.ListCancelableOrdersByUserAndSymbolParams{
			UserID: userID,
			Symbol: symbolA,
		},
	)
	require.NoError(t, err)

	scopedIDs := make([]int64, len(scoped))
	for i, o := range scoped {
		scopedIDs[i] = o.ID
	}

	require.ElementsMatch(
		t,
		[]int64{base + 1, base + 2, base + 3},
		scopedIDs,
		"symbol-scoped query must exclude cancelable orders on other symbols and non-cancelable orders on this one",
	)
}