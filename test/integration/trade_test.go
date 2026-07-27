package integration

import (
	"testing"
	"time"

	"velocity/internal/persistence/postgres/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCreateAndGetTrade(t *testing.T) {
	tc := NewTestContext(t)

    symbol := "BTCUSDT_" + uuid.New().String()[:8]
    
	//-----------------------------------
	// Create Symbol
	//-----------------------------------

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

	//-----------------------------------
	// Create Buyer
	//-----------------------------------

	buyerID := uuid.New()

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:           buyerID,
			Email:        buyerID.String() + "@test.com",
			PasswordHash: "hash",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	)
	require.NoError(t, err)

	//-----------------------------------
	// Create Seller
	//-----------------------------------

	sellerID := uuid.New()

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:           sellerID,
			Email:        sellerID.String() + "@test.com",
			PasswordHash: "hash",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	)
	require.NoError(t, err)

	//-----------------------------------
	// Create Buy Order
	//-----------------------------------

	buyOrderID := uuid.New()

	_, err = tc.OrderRepo.Create(
		tc.Ctx,
		generated.CreateOrderParams{
			ID:          buyOrderID,
			UserID:      buyerID,
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
			Quantity:  1,
			Remaining: 1,
			Filled:    0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	//-----------------------------------
	// Create Sell Order
	//-----------------------------------

	sellOrderID := uuid.New()

	_, err = tc.OrderRepo.Create(
		tc.Ctx,
		generated.CreateOrderParams{
			ID:          sellOrderID,
			UserID:      sellerID,
			Symbol:      symbol,
			Side:        "SELL",
			OrderType:   "LIMIT",
			TimeInForce: "GTC",
			Status:      "OPEN",
			Price: pgtype.Int8{
				Int64: 50000,
				Valid: true,
			},
			StopPrice: 0,
			Quantity:  1,
			Remaining: 1,
			Filled:    0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	//-----------------------------------
	// Create Trade
	//-----------------------------------

	tradeID := uuid.New()

	trade, err := tc.TradeRepo.Create(
		tc.Ctx,
		generated.CreateTradeParams{
			ID:          tradeID,
			BuyOrderID:  buyOrderID,
			SellOrderID: sellOrderID,
			BuyerID:     buyerID,
			SellerID:    sellerID,
			Symbol:      symbol,
			Price:       50000,
			Quantity:    1,
			ExecutedAt:  time.Now(),
		},
	)

	require.NoError(t, err)

	require.Equal(t, tradeID, trade.ID)

	//-----------------------------------
	// Get By ID
	//-----------------------------------

	dbTrade, err := tc.TradeRepo.GetByID(
		tc.Ctx,
		tradeID,
	)

	require.NoError(t, err)

	require.Equal(t, trade.ID, dbTrade.ID)
	require.Equal(t, trade.Symbol, dbTrade.Symbol)
	require.Equal(t, trade.Price, dbTrade.Price)
	require.Equal(t, trade.Quantity, dbTrade.Quantity)

	//-----------------------------------
	// Exists
	//-----------------------------------

	exists, err := tc.TradeRepo.TradeExists(
		tc.Ctx,
		tradeID,
	)

	require.NoError(t, err)
	require.True(t, exists)

	//-----------------------------------
	// List By User
	//-----------------------------------

	trades, err := tc.TradeRepo.ListByUser(
		tc.Ctx,
		buyerID,
	)

	require.NoError(t, err)
	require.NotEmpty(t, trades)

	//-----------------------------------
	// List By Symbol
	//-----------------------------------

	symbolTrades, err := tc.TradeRepo.ListBySymbol(
		tc.Ctx,
		symbol,
	)

	require.NoError(t, err)
	require.NotEmpty(t, symbolTrades)
}
