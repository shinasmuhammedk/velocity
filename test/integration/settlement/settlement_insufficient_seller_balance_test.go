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

func TestSettlement_InsufficientSellerBalance(t *testing.T) {
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

	//----------------------------------------------------------------------
	// Users
	//----------------------------------------------------------------------

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:           buyerID,
			Email:        uuid.NewString() + "@buyer.com",
			// PasswordHash: "password",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	)
	require.NoError(t, err)

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:           sellerID,
			Email:        uuid.NewString() + "@seller.com",
			// PasswordHash: "password",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	)
	require.NoError(t, err)

	//----------------------------------------------------------------------
	// Wallets
	//----------------------------------------------------------------------

	// Buyer has enough funds

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

	// Seller intentionally has insufficient BTC

	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID:    sellerID,
			Asset:     "BTC",
			Available: 0,
			Locked:    0, // Needs 1 BTC
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

	//----------------------------------------------------------------------
	// Orders
	//----------------------------------------------------------------------

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

	//----------------------------------------------------------------------
	// Settlement should FAIL
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

	require.Error(t, err)

	//----------------------------------------------------------------------
	// Trade should not exist
	//----------------------------------------------------------------------

	_, err = tc.TradeRepo.GetByID(tc.Ctx, tradeID)
	require.Error(t, err)

	//----------------------------------------------------------------------
	// Wallets unchanged
	//----------------------------------------------------------------------

	buyerUSDT, _ := tc.WalletRepo.Get(tc.Ctx, buyerID, "USDT")
	buyerBTC, _ := tc.WalletRepo.Get(tc.Ctx, buyerID, "BTC")

	require.EqualValues(t, 50000, buyerUSDT.Locked)
	require.EqualValues(t, 0, buyerBTC.Available)

	sellerBTC, _ := tc.WalletRepo.Get(tc.Ctx, sellerID, "BTC")
	sellerUSDT, _ := tc.WalletRepo.Get(tc.Ctx, sellerID, "USDT")

	require.EqualValues(t, 0, sellerBTC.Locked)
	require.EqualValues(t, 0, sellerUSDT.Available)

	//----------------------------------------------------------------------
	// Orders unchanged
	//----------------------------------------------------------------------

	buyOrder, _ := tc.OrderRepo.GetByID(tc.Ctx, buyOrderID)
	sellOrder, _ := tc.OrderRepo.GetByID(tc.Ctx, sellOrderID)

	require.EqualValues(t, 1, buyOrder.Remaining)
	require.EqualValues(t, 0, buyOrder.Filled)
	require.Equal(t, string(constants.OrderStatusOpen), buyOrder.Status)

	require.EqualValues(t, 1, sellOrder.Remaining)
	require.EqualValues(t, 0, sellOrder.Filled)
	require.Equal(t, string(constants.OrderStatusOpen), sellOrder.Status)

	//----------------------------------------------------------------------
	// No positions created
	//----------------------------------------------------------------------

	_, err = tc.PositionRepo.Get(tc.Ctx, buyerID, symbol)
	require.Error(t, err)

	_, err = tc.PositionRepo.Get(tc.Ctx, sellerID, symbol)
	require.Error(t, err)
}