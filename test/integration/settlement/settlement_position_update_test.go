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

func TestSettlement_PositionUpdate(t *testing.T) {
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
	// Each loop iteration below claims its own block of 10 IDs off this
	// base so buy/sell order and trade IDs never collide across iterations.
	base := time.Now().UnixNano()
	buyerID := base
	sellerID := base + 1

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
			Locked:    100000,
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
			Locked:    2,
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
	// Execute TWO settlements
	//----------------------------------------------------------------------

	for i := 0; i < 2; i++ {

		buyOrderID := base + 2 + int64(i)*10
		sellOrderID := base + 3 + int64(i)*10
		tradeID := base + 4 + int64(i)*10

		_, err = tc.OrderRepo.Create(
			tc.Ctx,
			generated.CreateOrderParams{
				ID:          buyOrderID,
				UserID:      buyerID,
				Symbol:      symbol,
				Side:        "BUY",
				OrderType:   "LIMIT",
				TimeInForce: "GTC",
				Status:      string(constants.OrderStatusOpen),
				Price:       price,
				Quantity:    1,
				Remaining:   1,
				Filled:      0,
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
				Status:      string(constants.OrderStatusOpen),
				Price:       price,
				Quantity:    1,
				Remaining:   1,
				Filled:      0,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		)
		require.NoError(t, err)

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

		require.NoError(t, err)
	}

	//----------------------------------------------------------------------
	// Positions should ACCUMULATE
	//----------------------------------------------------------------------

	buyerPosition, err := tc.PositionRepo.Get(
		tc.Ctx,
		buyerID,
		symbol,
	)
	require.NoError(t, err)

	sellerPosition, err := tc.PositionRepo.Get(
		tc.Ctx,
		sellerID,
		symbol,
	)
	require.NoError(t, err)

	require.EqualValues(t, 2, buyerPosition.Quantity)
	require.EqualValues(t, -2, sellerPosition.Quantity)
}
