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

func TestSettlement_WalletAccumulation(t *testing.T) {
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

	//----------------------------------------------------------------------
	// Users
	//----------------------------------------------------------------------

	buyerID := uuid.New()
	sellerID := uuid.New()

	users := []generated.CreateUserParams{
		{
			ID: buyerID,
			Email: uuid.NewString() + "@buyer.com",
			PasswordHash: "password",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID: sellerID,
			Email: uuid.NewString() + "@seller.com",
			PasswordHash: "password",
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
			UserID: buyerID,
			Asset: "USDT",
			Available: 0,
			Locked: 100000,
		},
		{
			UserID: buyerID,
			Asset: "BTC",
			Available: 0,
			Locked: 0,
		},
		{
			UserID: sellerID,
			Asset: "BTC",
			Available: 0,
			Locked: 2,
		},
		{
			UserID: sellerID,
			Asset: "USDT",
			Available: 0,
			Locked: 0,
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

		buyOrderID := uuid.New()
		sellOrderID := uuid.New()

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
				TradeID: uuid.New(),

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
	// Buyer Wallet
	//----------------------------------------------------------------------

	buyerBTC, err := tc.WalletRepo.Get(tc.Ctx, buyerID, "BTC")
	require.NoError(t, err)

	buyerUSDT, err := tc.WalletRepo.Get(tc.Ctx, buyerID, "USDT")
	require.NoError(t, err)

	require.EqualValues(t, 2, buyerBTC.Available)
	require.EqualValues(t, 0, buyerBTC.Locked)

	require.EqualValues(t, 0, buyerUSDT.Available)
	require.EqualValues(t, 0, buyerUSDT.Locked)

	//----------------------------------------------------------------------
	// Seller Wallet
	//----------------------------------------------------------------------

	sellerBTC, err := tc.WalletRepo.Get(tc.Ctx, sellerID, "BTC")
	require.NoError(t, err)

	sellerUSDT, err := tc.WalletRepo.Get(tc.Ctx, sellerID, "USDT")
	require.NoError(t, err)

	require.EqualValues(t, 0, sellerBTC.Available)
	require.EqualValues(t, 0, sellerBTC.Locked)

	require.EqualValues(t, 100000, sellerUSDT.Available)
	require.EqualValues(t, 0, sellerUSDT.Locked)
}