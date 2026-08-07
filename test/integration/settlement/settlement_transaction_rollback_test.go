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

func TestSettlement_Rollback(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := settlementservice.New(
		tc.TxManager,
		tc.UserDispatcher,
	)

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

	// Base for generating unique int64 IDs across repeated test runs
	// against a real database (uuid.New() previously served this role).
	base := time.Now().UnixNano()
	buyerID := base
	sellerID := base + 1
	buyOrderID := base + 2
	sellOrderID := base + 3
	tradeID := base + 4

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

	// Buyer wallets
	_, _ = tc.WalletRepo.Create(tc.Ctx, generated.CreateWalletParams{
		UserID: buyerID,
		Asset:  "USDT",
		Locked: 50000,
	})

	_, _ = tc.WalletRepo.Create(tc.Ctx, generated.CreateWalletParams{
		UserID: buyerID,
		Asset:  "BTC",
	})

	// Seller ONLY has USDT wallet.
	// BTC wallet is intentionally NOT created to force failure.

	_, _ = tc.WalletRepo.Create(tc.Ctx, generated.CreateWalletParams{
		UserID: sellerID,
		Asset:  "USDT",
	})

	price := pgtype.Int8{
		Int64: 50000,
		Valid: true,
	}

	_, _ = tc.OrderRepo.Create(tc.Ctx, generated.CreateOrderParams{
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
	})

	_, _ = tc.OrderRepo.Create(tc.Ctx, generated.CreateOrderParams{
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
	})

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

	//--------------------------------------------------
	// Verify NO trade persisted
	//--------------------------------------------------

	_, err = tc.TradeRepo.GetByID(tc.Ctx, tradeID)
	require.Error(t, err)

	//--------------------------------------------------
	// Buyer wallet unchanged
	//--------------------------------------------------

	buyerUSDT, _ := tc.WalletRepo.Get(tc.Ctx, buyerID, "USDT")

	require.EqualValues(t, 50000, buyerUSDT.Locked)
	require.EqualValues(t, 0, buyerUSDT.Available)

	//--------------------------------------------------
	// Orders unchanged
	//--------------------------------------------------

	buyOrder, _ := tc.OrderRepo.GetByID(tc.Ctx, buyOrderID)

	require.EqualValues(t, 1, buyOrder.Remaining)
	require.EqualValues(t, 0, buyOrder.Filled)
	require.Equal(
		t,
		string(constants.OrderStatusOpen),
		buyOrder.Status,
	)
}