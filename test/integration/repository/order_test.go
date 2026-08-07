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

func TestOrderRepository(t *testing.T) {

	tc := integration.NewTestContext(t)

	// Base for generating unique int64 IDs across repeated test runs
	// against a real database (uuid.New() previously served this role).
	base := time.Now().UnixNano()
	userID := base
	orderID := base + 1

	// ---------- Create User ----------
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

	// ---------- Create Symbol ----------
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

	// ---------- Create Order ----------
	order, err := tc.OrderRepo.Create(
		tc.Ctx,
		generated.CreateOrderParams{
			ID:          orderID,
			UserID:      userID,
			Symbol:      symbol,
			Side:        "BUY",
			OrderType:   "LIMIT",
			TimeInForce: "GTC",
			Status:      "OPEN",

			Price: pgtype.Int8{
				Int64: 50000,
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
	require.Equal(t, orderID, order.ID)

	// ---------- GetByID ----------
	dbOrder, err := tc.OrderRepo.GetByID(
		tc.Ctx,
		orderID,
	)

	require.NoError(t, err)
	require.Equal(t, order.ID, dbOrder.ID)

	// ---------- ListOpenOrders ----------
	openOrders, err := tc.OrderRepo.ListOpenOrders(
		tc.Ctx,
		symbol,
	)

	require.NoError(t, err)
	require.Len(t, openOrders, 1)
	require.Equal(t, orderID, openOrders[0].ID)

	// ---------- UpdateStatus ----------
	err = tc.OrderRepo.UpdateStatus(
		tc.Ctx,
		generated.UpdateOrderStatusParams{
			ID:     orderID,
			Status: "CANCELLED",
		},
	)

	require.NoError(t, err)

	dbOrder, err = tc.OrderRepo.GetByID(
		tc.Ctx,
		orderID,
	)

	require.NoError(t, err)
	require.Equal(t, "CANCELLED", dbOrder.Status)

	// ---------- UpdateOrderForModify ----------
	err = tc.OrderRepo.UpdateOrderForModify(
		tc.Ctx,
		generated.UpdateOrderForModifyParams{
			ID: orderID,
			Price: pgtype.Int8{
				Int64: 51000,
				Valid: true,
			},
			Quantity:  20,
			Remaining: 20,
		},
	)

	require.NoError(t, err)

	dbOrder, err = tc.OrderRepo.GetByID(
		tc.Ctx,
		orderID,
	)

	require.NoError(t, err)

	require.EqualValues(t, 20, dbOrder.Quantity)
	require.EqualValues(t, 20, dbOrder.Remaining)
	require.EqualValues(t, 51000, dbOrder.Price.Int64)

	// ---------- ListByUser ----------
	orders, err := tc.OrderRepo.ListByUser(
		tc.Ctx,
		userID,
	)

	require.NoError(t, err)
	require.NotEmpty(t, orders)

	// ---------- RecoveryOrders ----------
	recoveryOrders, err := tc.OrderRepo.RecoveryOrders(
		tc.Ctx,
	)

	require.NoError(t, err)
	require.NotNil(t, recoveryOrders)

	// ---------- Pending Stop Orders ----------
	stopOrders, err := tc.OrderRepo.GetPendingStopOrders(
		tc.Ctx,
	)

	require.NoError(t, err)
	require.NotNil(t, stopOrders)
}