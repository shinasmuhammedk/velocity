package integration

import (
	"testing"
	"time"

	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/service/settlementservice"
	"velocity/pkg/constants"
	testhelpers "velocity/test/helpers"
	"velocity/test/integration"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestSettlement_Success(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := settlementservice.New(tc.TxManager, tc.UserDispatcher)

	// ------------------------------------------------------------------
	// Create Symbol
	// ------------------------------------------------------------------

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
	require.NoError(t, err)

	// Base for generating unique int64 IDs across repeated test runs
	// against a real database (uuid.New() previously served this role).
	buyerID := testhelpers.NextID()
	sellerID := testhelpers.NextID()
	buyOrderID := testhelpers.NextID()
	sellOrderID := testhelpers.NextID()
	tradeID := testhelpers.NextID()

	// ------------------------------------------------------------------
	// Buyer
	// ------------------------------------------------------------------

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:    buyerID,
			Email: uuid.NewString() + "@buyer.com",
			// PasswordHash: "password",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	// ------------------------------------------------------------------
	// Seller
	// ------------------------------------------------------------------

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:    sellerID,
			Email: uuid.NewString() + "@seller.com",
			// PasswordHash: "password",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	// ------------------------------------------------------------------
	// Wallets
	// ------------------------------------------------------------------

	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID:    buyerID,
			Asset:     "USDT",
			Available: 0,
			Locked:    50000,
		},
	)
	require.NoError(t, err)

	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID:    buyerID,
			Asset:     "BTC",
			Available: 0,
			Locked:    0,
		},
	)
	require.NoError(t, err)

	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID:    sellerID,
			Asset:     "BTC",
			Available: 0,
			Locked:    1,
		},
	)
	require.NoError(t, err)

	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID:    sellerID,
			Asset:     "USDT",
			Available: 0,
			Locked:    0,
		},
	)
	require.NoError(t, err)

	// ------------------------------------------------------------------
	// Orders
	// ------------------------------------------------------------------

	price := pgtype.Int8{
		Int64: 50000,
		Valid: true,
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
			Status:      string(constants.OrderStatusOpen),
			Price:       price,
			StopPrice:   0,
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
			StopPrice:   0,
			Quantity:    1,
			Remaining:   1,
			Filled:      0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	)
	require.NoError(t, err)

	// ------------------------------------------------------------------
	// Settlement
	// ------------------------------------------------------------------

	err = service.Settle(
		tc.Ctx,
		settlementservice.SettlementRequest{
			TradeID:     tradeID,
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

	// ------------------------------------------------------------------
	// Buyer Wallet
	// ------------------------------------------------------------------

	buyerBTC, _ := tc.WalletRepo.Get(tc.Ctx, buyerID, "BTC")
	buyerUSDT, _ := tc.WalletRepo.Get(tc.Ctx, buyerID, "USDT")

	require.EqualValues(t, 1, buyerBTC.Available)
	require.EqualValues(t, 0, buyerUSDT.Locked)

	// ------------------------------------------------------------------
	// Seller Wallet
	// ------------------------------------------------------------------

	sellerBTC, _ := tc.WalletRepo.Get(tc.Ctx, sellerID, "BTC")
	sellerUSDT, _ := tc.WalletRepo.Get(tc.Ctx, sellerID, "USDT")

	require.EqualValues(t, 0, sellerBTC.Locked)
	require.EqualValues(t, 50000, sellerUSDT.Available)

	// ------------------------------------------------------------------
	// Positions
	// ------------------------------------------------------------------

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

	require.EqualValues(t, 1, buyerPosition.Quantity)
	require.EqualValues(t, -1, sellerPosition.Quantity)

	// ------------------------------------------------------------------
	// Trade persisted
	// ------------------------------------------------------------------

	trade, err := tc.TradeRepo.GetByID(tc.Ctx, tradeID)
	require.NoError(t, err)

	require.Equal(t, tradeID, trade.ID)

	// ------------------------------------------------------------------
	// Orders updated
	// ------------------------------------------------------------------

	buyOrder, err := tc.OrderRepo.GetByID(tc.Ctx, buyOrderID)
	require.NoError(t, err)

	sellOrder, err := tc.OrderRepo.GetByID(tc.Ctx, sellOrderID)
	require.NoError(t, err)

	require.EqualValues(t, 0, buyOrder.Remaining)
	require.EqualValues(t, 1, buyOrder.Filled)
	require.Equal(t, string(constants.OrderStatusFilled), buyOrder.Status)

	require.EqualValues(t, 0, sellOrder.Remaining)
	require.EqualValues(t, 1, sellOrder.Filled)
	require.Equal(t, string(constants.OrderStatusFilled), sellOrder.Status)
}
