package settlement

import (
	"testing"
	"time"

	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/service/settlementservice"
	"velocity/pkg/constants"
	"velocity/test/integration"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestSettlement_OrderAlreadyFilled(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := settlementservice.New(
		tc.TxManager,
		tc.UserDispatcher,
	)

	//----------------------------------------------------------------------
	// Symbol
	//----------------------------------------------------------------------

	symbol := "BTCUSDT-" + uuid.NewString()[:8]

	_, err := tc.SymbolRepo.Create(
		tc.Ctx,
		generated.CreateSymbolParams{
			Symbol:      symbol,
			DisplayName: "Bitcoin / Tether",
			TickSize:    1,
			LotSize:     1,
			IsActive:    true,
		},
	)
	require.NoError(t, err)

	// Base for generating unique int64 IDs across repeated test runs
	// against a real database (uuid.New() previously served this role).
	base := time.Now().UnixNano()
	buyerID := base
	sellerID := base + 1
	buyOrderID := base + 2
	sellOrderID := base + 3
	tradeID := base + 4
	nonExistentTradeID := base + 5

	//----------------------------------------------------------------------
	// Users
	//----------------------------------------------------------------------

	users := []generated.CreateUserParams{
		{
			ID:    buyerID,
			Email: uuid.NewString() + "@buyer.com",
			// PasswordHash: "password",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:    sellerID,
			Email: uuid.NewString() + "@seller.com",
			// PasswordHash: "password",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, u := range users {
		_, err := tc.UserRepo.Create(tc.Ctx, u)
		require.NoError(t, err)
	}

	//----------------------------------------------------------------------
	// Wallets
	//----------------------------------------------------------------------

	wallets := []generated.CreateWalletParams{
		{
			UserID:    buyerID,
			Asset:     "USDT",
			Available: 0,
			Locked:    50000,
		},
		{
			UserID:    buyerID,
			Asset:     "BTC",
			Available: 0,
			Locked:    0,
		},
		{
			UserID:    sellerID,
			Asset:     "BTC",
			Available: 0,
			Locked:    1,
		},
		{
			UserID:    sellerID,
			Asset:     "USDT",
			Available: 0,
			Locked:    0,
		},
	}

	for _, w := range wallets {
		_, err := tc.WalletRepo.Create(tc.Ctx, w)
		require.NoError(t, err)
	}

	price := pgtype.Int8{
		Int64: 50000,
		Valid: true,
	}

	//----------------------------------------------------------------------
	// Orders already FILLED
	//----------------------------------------------------------------------

	_, err = tc.OrderRepo.Create(
		tc.Ctx,
		generated.CreateOrderParams{
			ID:          buyOrderID,
			UserID:      buyerID,
			Symbol:      symbol,
			Side:        "BUY",
			OrderType:   "LIMIT",
			TimeInForce: "GTC",
			Status:      string(constants.OrderStatusFilled),
			Price:       price,
			Quantity:    1,
			Remaining:   0,
			Filled:      1,
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
			Status:      string(constants.OrderStatusFilled),
			Price:       price,
			Quantity:    1,
			Remaining:   0,
			Filled:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	)
	require.NoError(t, err)

	//----------------------------------------------------------------------
	// Settlement
	//----------------------------------------------------------------------

	err = service.Settle(
		tc.Ctx,
		settlementservice.SettlementRequest{
			TradeID: tradeID,

			BuyOrderID:  buyOrderID,
			SellOrderID: sellOrderID,

			BuyerID:  buyerID,
			SellerID: sellerID,

			Symbol: symbol,

			BaseAsset:  "BTC",
			QuoteAsset: "USDT",

			Price:    50000,
			Quantity: 1,
		},
	)

	//----------------------------------------------------------------------
	// EXPECTATION
	//----------------------------------------------------------------------

	require.Error(t, err)

	//----------------------------------------------------------------------
	// Orders should remain unchanged
	//----------------------------------------------------------------------

	buyOrder, err := tc.OrderRepo.GetByID(tc.Ctx, buyOrderID)
	require.NoError(t, err)

	sellOrder, err := tc.OrderRepo.GetByID(tc.Ctx, sellOrderID)
	require.NoError(t, err)

	require.Equal(t, string(constants.OrderStatusFilled), buyOrder.Status)
	require.EqualValues(t, 0, buyOrder.Remaining)
	require.EqualValues(t, 1, buyOrder.Filled)

	require.Equal(t, string(constants.OrderStatusFilled), sellOrder.Status)
	require.EqualValues(t, 0, sellOrder.Remaining)
	require.EqualValues(t, 1, sellOrder.Filled)

	//----------------------------------------------------------------------
	// No trade should be inserted
	//----------------------------------------------------------------------

	_, err = tc.TradeRepo.GetByID(tc.Ctx, nonExistentTradeID)
	require.Error(t, err)
}
