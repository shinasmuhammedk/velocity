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

func TestSettlement_PartialFill(t *testing.T) {
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
			DisplayName: "Bitcoin",
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

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID: buyerID,
			Email: uuid.NewString() + "@buyer.com",
			PasswordHash: "password",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID: sellerID,
			Email: uuid.NewString() + "@seller.com",
			PasswordHash: "password",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	//----------------------------------------------------------------------
	// Wallets
	//----------------------------------------------------------------------

	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID: buyerID,
			Asset: "USDT",
			Locked: 100000,
		},
	)
	require.NoError(t, err)

	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID: buyerID,
			Asset: "BTC",
		},
	)
	require.NoError(t, err)

	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID: sellerID,
			Asset: "BTC",
			Locked: 1,
		},
	)
	require.NoError(t, err)

	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID: sellerID,
			Asset: "USDT",
		},
	)
	require.NoError(t, err)

	//----------------------------------------------------------------------
	// Orders
	//----------------------------------------------------------------------

	price := pgtype.Int8{
		Int64: 50000,
		Valid: true,
	}

	buyOrderID := uuid.New()
	sellOrderID := uuid.New()

	// Buyer wants 2 BTC

	_, err = tc.OrderRepo.Create(
		tc.Ctx,
		generated.CreateOrderParams{
			ID: buyOrderID,
			UserID: buyerID,
			Symbol: symbol,
			Side: "BUY",
			OrderType: "LIMIT",
			TimeInForce: "GTC",
			Status: string(constants.OrderStatusOpen),
			Price: price,
			Quantity: 2,
			Remaining: 2,
			Filled: 0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	// Seller sells only 1 BTC

	_, err = tc.OrderRepo.Create(
		tc.Ctx,
		generated.CreateOrderParams{
			ID: sellOrderID,
			UserID: sellerID,
			Symbol: symbol,
			Side: "SELL",
			OrderType: "LIMIT",
			TimeInForce: "GTC",
			Status: string(constants.OrderStatusOpen),
			Price: price,
			Quantity: 1,
			Remaining: 1,
			Filled: 0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	//----------------------------------------------------------------------
	// Settlement
	//----------------------------------------------------------------------

	err = service.Settle(
		tc.Ctx,
		settlementservice.SettlementRequest{
			TradeID: uuid.New(),

			BuyOrderID: buyOrderID,
			SellOrderID: sellOrderID,

			BuyerID: buyerID,
			SellerID: sellerID,

			Symbol: symbol,

			BaseAsset: "BTC",
			QuoteAsset: "USDT",

			Price: 50000,
			Quantity: 1,
		},
	)

	require.NoError(t, err)

	//----------------------------------------------------------------------
	// Buyer Order
	//----------------------------------------------------------------------

	buyOrder, err := tc.OrderRepo.GetByID(tc.Ctx, buyOrderID)
	require.NoError(t, err)

	require.EqualValues(t, 1, buyOrder.Filled)
	require.EqualValues(t, 1, buyOrder.Remaining)
	require.Equal(t,
		string(constants.OrderStatusPartiallyFilled),
		buyOrder.Status,
	)

	//----------------------------------------------------------------------
	// Seller Order
	//----------------------------------------------------------------------

	sellOrder, err := tc.OrderRepo.GetByID(tc.Ctx, sellOrderID)
	require.NoError(t, err)

	require.EqualValues(t, 1, sellOrder.Filled)
	require.EqualValues(t, 0, sellOrder.Remaining)
	require.Equal(
		t,
		string(constants.OrderStatusFilled),
		sellOrder.Status,
	)

	//----------------------------------------------------------------------
	// Wallets
	//----------------------------------------------------------------------

	buyerBTC, _ := tc.WalletRepo.Get(tc.Ctx, buyerID, "BTC")
	sellerUSDT, _ := tc.WalletRepo.Get(tc.Ctx, sellerID, "USDT")

	require.EqualValues(t, 1, buyerBTC.Available)
	require.EqualValues(t, 50000, sellerUSDT.Available)

	//----------------------------------------------------------------------
	// Positions
	//----------------------------------------------------------------------

	buyerPos, err := tc.PositionRepo.Get(
		tc.Ctx,
		buyerID,
		symbol,
	)
	require.NoError(t, err)

	sellerPos, err := tc.PositionRepo.Get(
		tc.Ctx,
		sellerID,
		symbol,
	)
	require.NoError(t, err)

	require.EqualValues(t, 1, buyerPos.Quantity)
	require.EqualValues(t, -1, sellerPos.Quantity)
}