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

func TestSettlement_DuplicateTrade(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := settlementservice.New(
		tc.TxManager,
		tc.UserDispatcher,
	)

	// ----------------------------------------------------------------------
	// Symbol
	// ----------------------------------------------------------------------

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

	// ----------------------------------------------------------------------
	// Users
	// ----------------------------------------------------------------------

	base := time.Now().UnixNano()
	buyerID := base
	sellerID := base + 1

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

	// ----------------------------------------------------------------------
	// Wallets
	// ----------------------------------------------------------------------

	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID: buyerID,
			Asset:  "USDT",
			Locked: 50000,
		},
	)
	require.NoError(t, err)

	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID: buyerID,
			Asset:  "BTC",
		},
	)
	require.NoError(t, err)

	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID: sellerID,
			Asset:  "BTC",
			Locked: 1,
		},
	)
	require.NoError(t, err)

	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID: sellerID,
			Asset:  "USDT",
		},
	)
	require.NoError(t, err)

	// ----------------------------------------------------------------------
	// Orders
	// ----------------------------------------------------------------------

	price := pgtype.Int8{
		Int64: 50000,
		Valid: true,
	}

	buyOrderID := base + 2
	sellOrderID := base + 3

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
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	)
	require.NoError(t, err)

	// ----------------------------------------------------------------------
	// Settlement request
	// ----------------------------------------------------------------------

	tradeID := base + 4

	req := settlementservice.SettlementRequest{
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
	}

	// ----------------------------------------------------------------------
	// First settlement
	// ----------------------------------------------------------------------

	err = service.Settle(tc.Ctx, req)
	require.NoError(t, err)

	// ----------------------------------------------------------------------
	// Capture state
	// ----------------------------------------------------------------------

	buyerBTCBefore, _ := tc.WalletRepo.Get(tc.Ctx, buyerID, "BTC")
	buyerUSDTBefore, _ := tc.WalletRepo.Get(tc.Ctx, buyerID, "USDT")

	sellerBTCBefore, _ := tc.WalletRepo.Get(tc.Ctx, sellerID, "BTC")
	sellerUSDTBefore, _ := tc.WalletRepo.Get(tc.Ctx, sellerID, "USDT")

	// ----------------------------------------------------------------------
	// Call settlement AGAIN with same TradeID
	// ----------------------------------------------------------------------

	err = service.Settle(tc.Ctx, req)
	require.NoError(t, err)

	// ----------------------------------------------------------------------
	// Wallets should NOT change
	// ----------------------------------------------------------------------

	buyerBTCAfter, _ := tc.WalletRepo.Get(tc.Ctx, buyerID, "BTC")
	buyerUSDTAfter, _ := tc.WalletRepo.Get(tc.Ctx, buyerID, "USDT")

	sellerBTCAfter, _ := tc.WalletRepo.Get(tc.Ctx, sellerID, "BTC")
	sellerUSDTAfter, _ := tc.WalletRepo.Get(tc.Ctx, sellerID, "USDT")

	require.Equal(t, buyerBTCBefore.Available, buyerBTCAfter.Available)
	require.Equal(t, buyerUSDTBefore.Locked, buyerUSDTAfter.Locked)

	require.Equal(t, sellerBTCBefore.Locked, sellerBTCAfter.Locked)
	require.Equal(t, sellerUSDTBefore.Available, sellerUSDTAfter.Available)

	// ----------------------------------------------------------------------
	// Only ONE trade should exist
	// ----------------------------------------------------------------------

	trade, err := tc.TradeRepo.GetByID(tc.Ctx, tradeID)
	require.NoError(t, err)

	require.Equal(t, tradeID, trade.ID)

	// ----------------------------------------------------------------------
	// Orders should still be filled once
	// ----------------------------------------------------------------------

	buyOrder, err := tc.OrderRepo.GetByID(tc.Ctx, buyOrderID)
	require.NoError(t, err)

	sellOrder, err := tc.OrderRepo.GetByID(tc.Ctx, sellOrderID)
	require.NoError(t, err)

	require.EqualValues(t, 1, buyOrder.Filled)
	require.EqualValues(t, 0, buyOrder.Remaining)

	require.EqualValues(t, 1, sellOrder.Filled)
	require.EqualValues(t, 0, sellOrder.Remaining)
}
