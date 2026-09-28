package integration

import (
	"testing"
	"time"

	"velocity/internal/service/orderservice"
	"velocity/internal/persistence/postgres/generated"
	"velocity/pkg/errors"
	"velocity/test/integration"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// newOrderTradesTestService builds an orderservice.Service with only
// the dependencies GetOrderTrades actually touches (orderRepo,
// tradeRepo). It never reaches risk, wallet, the matching-engine
// registry, the dispatcher, or the id generator, so those are left nil.
func newOrderTradesTestService(tc *integration.TestContext) *orderservice.Service {
	return orderservice.New(
		tc.OrderRepo,
		tc.SymbolRepo,
		tc.UserRepo,
		tc.TradeRepo,
		nil, // risk
		nil, // wallet
		nil, // registry
		nil, // logger
		nil, // dispatcher
		nil, // idGenerator
	)
}

func TestGetOrderTrades_ReturnsOwnTrades(t *testing.T) {
	tc := integration.NewTestContext(t)
	svc := newOrderTradesTestService(tc)

	base := time.Now().UnixNano()
	buyerID := base
	sellerID := base + 1
	buyOrderID := base + 2
	sellOrderID := base + 3
	tradeID := base + 4

	symbol := "SYM_" + uuid.NewString()[:8]

	_, err := tc.SymbolRepo.Create(
		tc.Ctx,
		generated.CreateSymbolParams{
			Symbol:      symbol,
			DisplayName: symbol,
			TickSize:    1,
			LotSize:     1,
			IsActive:    true,
		},
	)
	require.NoError(t, err)

	for _, id := range []int64{buyerID, sellerID} {
		_, err := tc.UserRepo.Create(
			tc.Ctx,
			generated.CreateUserParams{
				ID:        id,
				Email:     uuid.NewString() + "@test.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		)
		require.NoError(t, err)
	}

	_, err = tc.OrderRepo.Create(
		tc.Ctx,
		generated.CreateOrderParams{
			ID:          buyOrderID,
			UserID:      buyerID,
			Symbol:      symbol,
			Side:        "BUY",
			OrderType:   "LIMIT",
			TimeInForce: "GTC",
			Status:      "FILLED",
			Price:       pgtype.Int8{Int64: 100, Valid: true},
			Quantity:    10,
			Remaining:   0,
			Filled:      10,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	)
	require.NoError(t, err)

	_, err = tc.OrderRepo.Create(
		tc.Ctx,
		generated.CreateOrderParams{
			ID:          sellOrderID,
			UserID:      sellerID,
			Symbol:      symbol,
			Side:        "SELL",
			OrderType:   "LIMIT",
			TimeInForce: "GTC",
			Status:      "FILLED",
			Price:       pgtype.Int8{Int64: 100, Valid: true},
			Quantity:    10,
			Remaining:   0,
			Filled:      10,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	)
	require.NoError(t, err)

	_, err = tc.TradeRepo.Create(
		tc.Ctx,
		generated.CreateTradeParams{
			ID:          tradeID,
			BuyOrderID:  buyOrderID,
			SellOrderID: sellOrderID,
			BuyerID:     buyerID,
			SellerID:    sellerID,
			Symbol:      symbol,
			Price:       100,
			Quantity:    10,
			ExecutedAt:  time.Now(),
		},
	)
	require.NoError(t, err)

	// The buyer asking for the fills on their own order gets them back.
	trades, err := svc.GetOrderTrades(tc.Ctx, buyOrderID, buyerID)
	require.NoError(t, err)
	require.Len(t, trades, 1)
	require.Equal(t, tradeID, trades[0].ID)

	// Same for the seller on their own order.
	trades, err = svc.GetOrderTrades(tc.Ctx, sellOrderID, sellerID)
	require.NoError(t, err)
	require.Len(t, trades, 1)
	require.Equal(t, tradeID, trades[0].ID)
}

func TestGetOrderTrades_RejectsNonOwner(t *testing.T) {
	tc := integration.NewTestContext(t)
	svc := newOrderTradesTestService(tc)

	base := time.Now().UnixNano()
	ownerID := base
	attackerID := base + 1
	orderID := base + 2

	symbol := "SYM_" + uuid.NewString()[:8]

	_, err := tc.SymbolRepo.Create(
		tc.Ctx,
		generated.CreateSymbolParams{
			Symbol:      symbol,
			DisplayName: symbol,
			TickSize:    1,
			LotSize:     1,
			IsActive:    true,
		},
	)
	require.NoError(t, err)

	for _, id := range []int64{ownerID, attackerID} {
		_, err := tc.UserRepo.Create(
			tc.Ctx,
			generated.CreateUserParams{
				ID:        id,
				Email:     uuid.NewString() + "@test.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		)
		require.NoError(t, err)
	}

	_, err = tc.OrderRepo.Create(
		tc.Ctx,
		generated.CreateOrderParams{
			ID:          orderID,
			UserID:      ownerID,
			Symbol:      symbol,
			Side:        "BUY",
			OrderType:   "LIMIT",
			TimeInForce: "GTC",
			Status:      "OPEN",
			Price:       pgtype.Int8{Int64: 100, Valid: true},
			Quantity:    10,
			Remaining:   10,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	)
	require.NoError(t, err)

	// A user who does not own the order must not be able to fetch its
	// trades - and the error must look identical to "order not found"
	// rather than revealing that an order with this ID exists.
	trades, err := svc.GetOrderTrades(tc.Ctx, orderID, attackerID)

	require.ErrorIs(t, err, errors.ErrOrderNotFound)
	require.Nil(t, trades)
}

func TestGetOrderTrades_UnknownOrder(t *testing.T) {
	tc := integration.NewTestContext(t)
	svc := newOrderTradesTestService(tc)

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

	trades, err := svc.GetOrderTrades(tc.Ctx, time.Now().UnixNano()+999, userID)

	require.ErrorIs(t, err, errors.ErrOrderNotFound)
	require.Nil(t, trades)
}