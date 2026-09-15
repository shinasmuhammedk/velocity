package worker

import (
	"context"
	"testing"
	"time"

	"velocity/internal/domain/trade"
	"velocity/internal/engine"
	"velocity/internal/engine/orderbook"
	"velocity/internal/marketdata"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/worker"
	"velocity/internal/service/settlementservice"
	"velocity/pkg/constants"
	testhelpers "velocity/test/helpers"
	"velocity/test/integration"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestTradeConsumer_SettlementFailureRecorded(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := settlementservice.New(
		tc.TxManager,
		tc.UserDispatcher,
	)

	// ---------------------------------------------------------------------
	// Symbol
	// ---------------------------------------------------------------------

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

	// ---------------------------------------------------------------------
	// IDs
	// ---------------------------------------------------------------------

	base := time.Now().UnixNano()

	buyerID := base
	sellerID := base + 1
	buyOrderID := base + 2
	sellOrderID := base + 3
	tradeID := base + 4

	// ---------------------------------------------------------------------
	// Users
	// ---------------------------------------------------------------------

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:        buyerID,
			Email:     uuid.NewString() + "@buyer.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:        sellerID,
			Email:     uuid.NewString() + "@seller.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	// ---------------------------------------------------------------------
	// Wallets
	//
	// Buyer deliberately has no locked USDT.
	// The trade requires 50,000 USDT, so settlement must fail.
	// ---------------------------------------------------------------------

	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID: buyerID,
			Asset:  "USDT",
			Locked: 0,
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

	// ---------------------------------------------------------------------
	// Orders
	// ---------------------------------------------------------------------

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

	// ---------------------------------------------------------------------
	// Consumer dependencies
	//
	// The failure path never calls DispatchTrade, so a broadcaster with
	// nil dependencies is sufficient for this specific test.
	// ---------------------------------------------------------------------

	logger := zap.NewNop()

	dispatcher := marketdata.NewBroadcaster(
		nil,
		nil,
	)

	provider := func(string) *orderbook.OrderBook {
		return nil
	}

	consumer := worker.NewTradeConsumer(
		service,
		tc.SymbolRepo,
		tc.FailedSettlementRepo,
		dispatcher,
		provider,
		logger,
	)

	// ---------------------------------------------------------------------
	// Start consumer
	// ---------------------------------------------------------------------

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trades := make(chan trade.Trade, 1)

	consumer.Start(ctx, trades)

	// ---------------------------------------------------------------------
	// Submit trade
	// ---------------------------------------------------------------------

	trades <- trade.Trade{
		ID:          tradeID,
		BuyOrderID:  buyOrderID,
		SellOrderID: sellOrderID,
		BuyerID:     buyerID,
		SellerID:    sellerID,
		Symbol:      symbol,
		Price:       50000,
		Quantity:    1,
		ExecutedAt:  time.Now(),
	}

	// ---------------------------------------------------------------------
	// Wait for TradeConsumer to record the failed settlement.
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	// Wait for TradeConsumer to record the failed settlement.
	// ---------------------------------------------------------------------

	// ---------------------------------------------------------------------
	// Wait for TradeConsumer to record the failed settlement.
	// ---------------------------------------------------------------------

	var failed generated.FailedSettlement

	require.Eventually(
		t,
		func() bool {
			failedSettlements, err := tc.FailedSettlementRepo.ListUnresolved(tc.Ctx)
			if err != nil {
				t.Logf("failed settlements lookup failed: %v", err)
				return false
			}

			for _, current := range failedSettlements {
				if current.TradeID == tradeID {
					failed = current
					return true
				}
			}

			return false
		},
		2*time.Second,
		10*time.Millisecond,
		"failed settlement was not recorded",
	)

	// ---------------------------------------------------------------------
	// Verify failed settlement record.
	// ---------------------------------------------------------------------

	require.Equal(t, tradeID, failed.TradeID)
	require.Equal(t, buyOrderID, failed.BuyOrderID)
	require.Equal(t, sellOrderID, failed.SellOrderID)
	require.Equal(t, buyerID, failed.BuyerID)
	require.Equal(t, sellerID, failed.SellerID)
	require.Equal(t, symbol, failed.Symbol)
	require.Equal(t, int64(50000), failed.Price)
	require.Equal(t, int64(1), failed.Quantity)

	require.NotEmpty(t, failed.ErrorMessage)
	require.Equal(t, int32(0), failed.RetryCount)
	require.False(t, failed.Resolved)
	require.False(t, failed.IsDead)

	// ---------------------------------------------------------------------
	// Verify settlement transaction rolled back.
	// ---------------------------------------------------------------------

	_, err = tc.TradeRepo.GetByID(
		tc.Ctx,
		tradeID,
	)
	require.Error(t, err)

	buyOrder, err := tc.OrderRepo.GetByID(
		tc.Ctx,
		buyOrderID,
	)
	require.NoError(t, err)

	require.EqualValues(t, 0, buyOrder.Filled)
	require.EqualValues(t, 1, buyOrder.Remaining)
	require.Equal(
		t,
		string(constants.OrderStatusOpen),
		buyOrder.Status,
	)

	sellOrder, err := tc.OrderRepo.GetByID(
		tc.Ctx,
		sellOrderID,
	)
	require.NoError(t, err)

	require.EqualValues(t, 0, sellOrder.Filled)
	require.EqualValues(t, 1, sellOrder.Remaining)
	require.Equal(
		t,
		string(constants.OrderStatusOpen),
		sellOrder.Status,
	)

	buyerUSDT, err := tc.WalletRepo.Get(
		tc.Ctx,
		buyerID,
		"USDT",
	)
	require.NoError(t, err)

	require.EqualValues(t, 0, buyerUSDT.Available)
	require.EqualValues(t, 0, buyerUSDT.Locked)

	sellerBTC, err := tc.WalletRepo.Get(
		tc.Ctx,
		sellerID,
		"BTC",
	)
	require.NoError(t, err)

	require.EqualValues(t, 0, sellerBTC.Available)
	require.EqualValues(t, 1, sellerBTC.Locked)
}

func TestTradeConsumer_SettlementSuccess(t *testing.T) {
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
			DisplayName: "Bitcoin / Tether",
			TickSize:    1,
			LotSize:     1,
			IsActive:    true,
			BaseAsset:   "BTC",
			QuoteAsset:  "USDT",
		},
	)
	require.NoError(t, err)

	buyerID := time.Now().UnixNano()
	sellerID := buyerID + 1
	buyOrderID := buyerID + 2
	sellOrderID := buyerID + 3
	tradeID := buyerID + 4

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:        buyerID,
			Email:     uuid.NewString() + "@buyer.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:        sellerID,
			Email:     uuid.NewString() + "@seller.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

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

	logger := zap.NewNop()

	dispatcher := marketdata.NewBroadcaster(
		nil,
		nil,
	)

	// Settlement happens before orderBookFor is used.
	// Returning nil here is therefore sufficient to test
	// the successful settlement path.
	provider := func(string) *orderbook.OrderBook {
		return nil
	}

	consumer := worker.NewTradeConsumer(
		service,
		tc.SymbolRepo,
		tc.FailedSettlementRepo,
		dispatcher,
		provider,
		logger,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trades := make(chan trade.Trade, 1)

	consumer.Start(ctx, trades)

	trades <- trade.Trade{
		ID:          tradeID,
		BuyOrderID:  buyOrderID,
		SellOrderID: sellOrderID,
		BuyerID:     buyerID,
		SellerID:    sellerID,
		Symbol:      symbol,
		Price:       50000,
		Quantity:    1,
		ExecutedAt:  time.Now(),
	}

	require.Eventually(
		t,
		func() bool {
			settledTrade, err := tc.TradeRepo.GetByID(
				tc.Ctx,
				tradeID,
			)

			return err == nil && settledTrade.ID == tradeID
		},
		2*time.Second,
		10*time.Millisecond,
	)

	buyOrder, err := tc.OrderRepo.GetByID(
		tc.Ctx,
		buyOrderID,
	)
	require.NoError(t, err)

	sellOrder, err := tc.OrderRepo.GetByID(
		tc.Ctx,
		sellOrderID,
	)
	require.NoError(t, err)

	require.EqualValues(t, 0, buyOrder.Remaining)
	require.EqualValues(t, 1, buyOrder.Filled)
	require.Equal(
		t,
		string(constants.OrderStatusFilled),
		buyOrder.Status,
	)

	require.EqualValues(t, 0, sellOrder.Remaining)
	require.EqualValues(t, 1, sellOrder.Filled)
	require.Equal(
		t,
		string(constants.OrderStatusFilled),
		sellOrder.Status,
	)

	buyerBTC, err := tc.WalletRepo.Get(
		tc.Ctx,
		buyerID,
		"BTC",
	)
	require.NoError(t, err)

	buyerUSDT, err := tc.WalletRepo.Get(
		tc.Ctx,
		buyerID,
		"USDT",
	)
	require.NoError(t, err)

	require.EqualValues(t, 1, buyerBTC.Available)
	require.EqualValues(t, 0, buyerUSDT.Locked)

	sellerBTC, err := tc.WalletRepo.Get(
		tc.Ctx,
		sellerID,
		"BTC",
	)
	require.NoError(t, err)

	sellerUSDT, err := tc.WalletRepo.Get(
		tc.Ctx,
		sellerID,
		"USDT",
	)
	require.NoError(t, err)

	require.EqualValues(t, 0, sellerBTC.Locked)
	require.EqualValues(t, 50000, sellerUSDT.Available)

}

func TestEngineTradeReachesTradeConsumerAndSettles(t *testing.T) {
	tc := integration.NewTestContext(t)

	tc.CleanupTrades(t)

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

	buyerID := testhelpers.NextID()
	sellerID := testhelpers.NextID()

	buyOrderID := testhelpers.NextID()
	sellOrderID := testhelpers.NextID()
	// tradeID := testhelpers.NextID()

	// Create buyer and seller.
	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:        buyerID,
			Email:     uuid.NewString() + "@buyer.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:        sellerID,
			Email:     uuid.NewString() + "@seller.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	// Buyer has 50,000 USDT locked.
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

	// Seller has 1 BTC locked.
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

	// These orders exist in DB because SettlementService
	// updates the existing orders after the engine generates
	// the trade.
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
			Price:       pgtype.Int8{Int64: 50000, Valid: true},
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
			Price:       pgtype.Int8{Int64: 50000, Valid: true},
			StopPrice:   0,
			Quantity:    1,
			Remaining:   1,
			Filled:      0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	)
	require.NoError(t, err)

	// Build the actual engine.
	e := engine.New(symbol, nil, nil)

	// The engine must be stopped when the test finishes.
	t.Cleanup(func() {
		e.Stop()
	})

	service := settlementservice.New(
		tc.TxManager,
		tc.UserDispatcher,
	)

	dispatcher := marketdata.NewBroadcaster(
		nil,
		nil,
	)

	// We are testing settlement here. Market-data broadcasting
	// is deliberately not part of this test.
	provider := func(string) *orderbook.OrderBook {
		return nil
	}

	consumer := worker.NewTradeConsumer(
		service,
		tc.SymbolRepo,
		tc.FailedSettlementRepo,
		dispatcher,
		provider,
		zap.NewExample(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// THIS IS THE IMPORTANT CONNECTION:
	//
	// Actual Engine trade queue
	//          ↓
	//      TradeConsumer
	//
	consumer.Start(ctx, e.Trades())

	// Put the BUY order into the engine first.
	buy := testhelpers.NewOrder(
		buyOrderID,
		buyerID,
		constants.OrderSideBuy,
		50000,
		1,
	)

	err = e.SubmitOrder(buy)
	require.NoError(t, err)

	// Then submit the SELL order.
	//
	// This should match the existing BUY order and cause
	// the engine to generate a real trade.
	sell := testhelpers.NewOrder(
		sellOrderID,
		sellerID,
		constants.OrderSideSell,
		50000,
		1,
	)

	err = e.SubmitOrder(sell)
	require.NoError(t, err)

	t.Logf(
		"ENGINE BUY: status=%s remaining=%d filled=%d",
		buy.Status,
		buy.Remaining,
		buy.Filled,
	)

	t.Logf(
		"ENGINE SELL: status=%s remaining=%d filled=%d",
		sell.Status,
		sell.Remaining,
		sell.Filled,
	)

	t.Logf(
		"ENGINE LAST TRADE PRICE: %d",
		e.LastTradePrice(),
	)

	t.Logf(
		"ENGINE BEST BID: %d",
		e.OrderBook().BestBidPrice(),
	)

	t.Logf(
		"ENGINE BEST ASK: %d",
		e.OrderBook().BestAskPrice(),
	)

	// The TradeConsumer should receive the trade generated
	// by the actual engine and settle it.
	require.Eventually(
		t,
		func() bool {
			buyOrder, buyErr := tc.OrderRepo.GetByID(tc.Ctx, buyOrderID)
			if buyErr != nil {
				t.Logf("buy order lookup failed: %v", buyErr)
				return false
			}

			sellOrder, sellErr := tc.OrderRepo.GetByID(tc.Ctx, sellOrderID)
			if sellErr != nil {
				t.Logf("sell order lookup failed: %v", sellErr)
				return false
			}

			t.Logf(
				"BUY order: status=%s remaining=%d filled=%d",
				buyOrder.Status,
				buyOrder.Remaining,
				buyOrder.Filled,
			)

			t.Logf(
				"SELL order: status=%s remaining=%d filled=%d",
				sellOrder.Status,
				sellOrder.Remaining,
				sellOrder.Filled,
			)

			return buyOrder.Remaining == 0 &&
				buyOrder.Filled == 1 &&
				buyOrder.Status == string(constants.OrderStatusFilled) &&
				sellOrder.Remaining == 0 &&
				sellOrder.Filled == 1 &&
				sellOrder.Status == string(constants.OrderStatusFilled)
		},
		5*time.Second,
		100*time.Millisecond,
	)

	// Verify DB orders.
	buyOrder, err := tc.OrderRepo.GetByID(tc.Ctx, buyOrderID)
	require.NoError(t, err)

	sellOrder, err := tc.OrderRepo.GetByID(tc.Ctx, sellOrderID)
	require.NoError(t, err)

	require.EqualValues(t, 0, buyOrder.Remaining)
	require.EqualValues(t, 1, buyOrder.Filled)
	require.Equal(
		t,
		string(constants.OrderStatusFilled),
		buyOrder.Status,
	)

	require.EqualValues(t, 0, sellOrder.Remaining)
	require.EqualValues(t, 1, sellOrder.Filled)
	require.Equal(
		t,
		string(constants.OrderStatusFilled),
		sellOrder.Status,
	)

	// Verify buyer wallet.
	buyerBTC, err := tc.WalletRepo.Get(tc.Ctx, buyerID, "BTC")
	require.NoError(t, err)

	buyerUSDT, err := tc.WalletRepo.Get(tc.Ctx, buyerID, "USDT")
	require.NoError(t, err)

	require.EqualValues(t, 1, buyerBTC.Available)
	require.EqualValues(t, 0, buyerUSDT.Locked)

	// Verify seller wallet.
	sellerBTC, err := tc.WalletRepo.Get(tc.Ctx, sellerID, "BTC")
	require.NoError(t, err)

	sellerUSDT, err := tc.WalletRepo.Get(tc.Ctx, sellerID, "USDT")
	require.NoError(t, err)

	require.EqualValues(t, 0, sellerBTC.Locked)
	require.EqualValues(t, 50000, sellerUSDT.Available)
}
