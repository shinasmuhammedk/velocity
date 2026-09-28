package worker_test

import (
	"context"
	"testing"
	"time"

	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/persistence/worker"
	"velocity/internal/service/settlementservice"
	"velocity/pkg/constants"
	"velocity/test/integration"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestFailedSettlementWorker_RetrySuccess(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := settlementservice.New(
		tc.TxManager,
		tc.UserDispatcher,
	)

	worker := worker.NewFailedSettlementWorker(
		service,
		tc.FailedSettlementRepo,
		tc.SymbolRepo,
		zap.NewNop(),
		time.Second,
	)

	symbol := "BTCUSDT-" + uuid.NewString()[:8]

	_, err := tc.SymbolRepo.Create(
		tc.Ctx,
		generated.CreateSymbolParams{
			Symbol:      symbol,
			DisplayName: "Bitcoin",
			BaseAsset:   "BTC",
			QuoteAsset:  "USDT",
			TickSize:    1,
			LotSize:     1,
			IsActive:    true,
		},
	)
	require.NoError(t, err)

	buyerID := time.Now().UnixNano()
	sellerID := buyerID + 1

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:        buyerID,
			Email:     "buyer-" + uuid.NewString() + "@test.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:        sellerID,
			Email:     "seller-" + uuid.NewString() + "@test.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	// Buyer has the quote asset locked and ready to spend.
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

	// Seller has the base asset locked and ready to spend.
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

	buyOrderID := buyerID + 100
	sellOrderID := sellerID + 100

	price := pgtype.Int8{
		Int64: 50000,
		Valid: true,
	}

	now := time.Now()

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
			Price:       price,
			Quantity:    1,
			Remaining:   1,
			Filled:      0,
			CreatedAt:   now,
			UpdatedAt:   now,
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
			Status:      "OPEN",
			Price:       price,
			Quantity:    1,
			Remaining:   1,
			Filled:      0,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	)
	require.NoError(t, err)

	tradeID := buyOrderID + 1000

	failed, err := tc.FailedSettlementRepo.Create(
		tc.Ctx,
		generated.CreateFailedSettlementParams{
			TradeID:      tradeID,
			BuyOrderID:   buyOrderID,
			SellOrderID:  sellOrderID,
			BuyerID:      buyerID,
			SellerID:     sellerID,
			Symbol:       symbol,
			Price:        50000,
			Quantity:     1,
			ErrorMessage: "temporary settlement failure",
		},
	)
	require.NoError(t, err)

	require.Equal(t, int32(0), failed.RetryCount)
	require.False(t, failed.Resolved)
	require.False(t, failed.IsDead)

	ctx, cancel := context.WithCancel(tc.Ctx)
	defer cancel()

	worker.Start(ctx)

	require.Eventually(t, func() bool {
		resolved, err := tc.FailedSettlementRepo.Get(
			tc.Ctx,
			failed.ID,
		)
		if err != nil {
			return false
		}

		return resolved.Resolved
	}, 2*time.Second, 20*time.Millisecond)

	resolved, err := tc.FailedSettlementRepo.Get(
		tc.Ctx,
		failed.ID,
	)
	require.NoError(t, err)

	require.True(t, resolved.Resolved)
	require.False(t, resolved.IsDead)
	require.Equal(t, int32(0), resolved.RetryCount)

	// The retry must have created the trade.
	tradeRecord, err := tc.TradeRepo.GetByID(
		tc.Ctx,
		tradeID,
	)
	require.NoError(t, err)
	require.Equal(t, tradeID, tradeRecord.ID)
	require.Equal(t, buyOrderID, tradeRecord.BuyOrderID)
	require.Equal(t, sellOrderID, tradeRecord.SellOrderID)

	// Buyer spent locked USDT and received BTC.
	buyerUSDT, err := tc.WalletRepo.Get(
		tc.Ctx,
		buyerID,
		"USDT",
	)
	require.NoError(t, err)

	require.Equal(t, int64(0), buyerUSDT.Locked)
	require.Equal(t, int64(0), buyerUSDT.Available)

	buyerBTC, err := tc.WalletRepo.Get(
		tc.Ctx,
		buyerID,
		"BTC",
	)
	require.NoError(t, err)

	require.Equal(t, int64(1), buyerBTC.Available)
	require.Equal(t, int64(0), buyerBTC.Locked)

	// Seller spent locked BTC and received USDT.
	sellerBTC, err := tc.WalletRepo.Get(
		tc.Ctx,
		sellerID,
		"BTC",
	)
	require.NoError(t, err)

	require.Equal(t, int64(0), sellerBTC.Locked)
	require.Equal(t, int64(0), sellerBTC.Available)

	sellerUSDT, err := tc.WalletRepo.Get(
		tc.Ctx,
		sellerID,
		"USDT",
	)
	require.NoError(t, err)

	require.Equal(t, int64(50000), sellerUSDT.Available)
	require.Equal(t, int64(0), sellerUSDT.Locked)

	// Both orders must now be filled.
	buyOrder, err := tc.OrderRepo.GetByID(
		tc.Ctx,
		buyOrderID,
	)
	require.NoError(t, err)

	require.Equal(t, int64(0), buyOrder.Remaining)
	require.Equal(t, int64(1), buyOrder.Filled)
	require.Equal(t, "FILLED", buyOrder.Status)

	sellOrder, err := tc.OrderRepo.GetByID(
		tc.Ctx,
		sellOrderID,
	)
	require.NoError(t, err)

	require.Equal(t, int64(0), sellOrder.Remaining)
	require.Equal(t, int64(1), sellOrder.Filled)
	require.Equal(t, "FILLED", sellOrder.Status)
}

func TestFailedSettlementWorker_RetryFailure(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := settlementservice.New(
		tc.TxManager,
		tc.UserDispatcher,
	)

	worker := worker.NewFailedSettlementWorker(
		service,
		tc.FailedSettlementRepo,
		tc.SymbolRepo,
		zap.NewNop(),
		10*time.Millisecond,
	)

	symbol := "BTCUSDT-" + uuid.NewString()[:8]

	_, err := tc.SymbolRepo.Create(
		tc.Ctx,
		generated.CreateSymbolParams{
			Symbol:      symbol,
			DisplayName: "Bitcoin",
			BaseAsset:   "BTC",
			QuoteAsset:  "USDT",
			TickSize:    1,
			LotSize:     1,
			IsActive:    true,
		},
	)
	require.NoError(t, err)

	buyerID := time.Now().UnixNano()
	sellerID := buyerID + 1

	now := time.Now()

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:        buyerID,
			Email:     "buyer-" + uuid.NewString() + "@test.com",
			CreatedAt: now,
			UpdatedAt: now,
		},
	)
	require.NoError(t, err)

	_, err = tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:        sellerID,
			Email:     "seller-" + uuid.NewString() + "@test.com",
			CreatedAt: now,
			UpdatedAt: now,
		},
	)
	require.NoError(t, err)

	// Buyer deliberately does NOT have enough locked USDT.
	// Settlement must fail because it needs 50000 USDT.
	_, err = tc.WalletRepo.Create(
		tc.Ctx,
		generated.CreateWalletParams{
			UserID:    buyerID,
			Asset:     "USDT",
			Available: 0,
			Locked:    0,
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

	// Seller has the BTC required for the trade.
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

	buyOrderID := buyerID + 100
	sellOrderID := sellerID + 100

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
			Status:      "OPEN",
			Price:       price,
			Quantity:    1,
			Remaining:   1,
			Filled:      0,
			CreatedAt:   now,
			UpdatedAt:   now,
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
			Status:      "OPEN",
			Price:       price,
			Quantity:    1,
			Remaining:   1,
			Filled:      0,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	)
	require.NoError(t, err)

	tradeID := buyOrderID + 1000

	failed, err := tc.FailedSettlementRepo.Create(
		tc.Ctx,
		generated.CreateFailedSettlementParams{
			TradeID:      tradeID,
			BuyOrderID:   buyOrderID,
			SellOrderID:  sellOrderID,
			BuyerID:      buyerID,
			SellerID:     sellerID,
			Symbol:       symbol,
			Price:        50000,
			Quantity:     1,
			ErrorMessage: "temporary settlement failure",
		},
	)
	require.NoError(t, err)

	require.Equal(t, int32(0), failed.RetryCount)
	require.False(t, failed.Resolved)
	require.False(t, failed.IsDead)

	ctx, cancel := context.WithCancel(tc.Ctx)
	defer cancel()

	worker.Start(ctx)

	// Wait until multiple failed retry attempts have occurred.
	require.Eventually(t, func() bool {
		current, err := tc.FailedSettlementRepo.Get(
			tc.Ctx,
			failed.ID,
		)
		if err != nil {
			return false
		}

		return current.RetryCount >= 2
	}, 2*time.Second, 20*time.Millisecond)

	current, err := tc.FailedSettlementRepo.Get(
		tc.Ctx,
		failed.ID,
	)
	require.NoError(t, err)

	require.GreaterOrEqual(t, current.RetryCount, int32(2))
	require.False(t, current.Resolved)
	require.False(t, current.IsDead)
	require.Less(t, current.RetryCount, int32(10))

	// Settlement must have rolled back completely.
	_, err = tc.TradeRepo.GetByID(
		tc.Ctx,
		tradeID,
	)
	require.Error(t, err)

	// Orders must remain untouched.
	buyOrder, err := tc.OrderRepo.GetByID(
		tc.Ctx,
		buyOrderID,
	)
	require.NoError(t, err)

	require.Equal(t, int64(1), buyOrder.Remaining)
	require.Equal(t, int64(0), buyOrder.Filled)
	require.Equal(t, "OPEN", buyOrder.Status)

	sellOrder, err := tc.OrderRepo.GetByID(
		tc.Ctx,
		sellOrderID,
	)
	require.NoError(t, err)

	require.Equal(t, int64(1), sellOrder.Remaining)
	require.Equal(t, int64(0), sellOrder.Filled)
	require.Equal(t, "OPEN", sellOrder.Status)

	// Wallets must remain untouched.
	buyerUSDT, err := tc.WalletRepo.Get(
		tc.Ctx,
		buyerID,
		"USDT",
	)
	require.NoError(t, err)

	require.Equal(t, int64(0), buyerUSDT.Available)
	require.Equal(t, int64(0), buyerUSDT.Locked)

	sellerBTC, err := tc.WalletRepo.Get(
		tc.Ctx,
		sellerID,
		"BTC",
	)
	require.NoError(t, err)

	require.Equal(t, int64(0), sellerBTC.Available)
	require.Equal(t, int64(1), sellerBTC.Locked)
}

func TestFailedSettlementWorker_MarksDead(t *testing.T) {
	tc := integration.NewTestContext(t)

	settlement := settlementservice.New(
		tc.TxManager,
		tc.UserDispatcher,
	)

	worker := worker.NewFailedSettlementWorker(
		settlement,
		tc.FailedSettlementRepo,
		tc.SymbolRepo,
		zap.NewNop(),
		10*time.Millisecond,
	)

	now := time.Now()

	// Unique IDs for this test.
	buyerID := now.UnixNano()
	sellerID := buyerID + 1

	symbol := "BTCUSDT-" + uuid.NewString()[:8]

	// Create symbol.
	_, err := tc.SymbolRepo.Create(tc.Ctx, generated.CreateSymbolParams{
		Symbol:     symbol,
		BaseAsset:  "BTC",
		QuoteAsset: "USDT",
		TickSize:   1,
		LotSize:    1,
		IsActive:   true,
	})
	require.NoError(t, err)

	// Create users.
	_, err = tc.UserRepo.Create(tc.Ctx, generated.CreateUserParams{
		ID:        buyerID,
		Email:     uuid.NewString() + "@buyer.test",
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, err)

	_, err = tc.UserRepo.Create(tc.Ctx, generated.CreateUserParams{
		ID:        sellerID,
		Email:     uuid.NewString() + "@seller.test",
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, err)

	buyOrderID := buyerID + 100
	sellOrderID := sellerID + 100
	tradeID := buyOrderID + 1000

	// Create a failed settlement.
	failed, err := tc.FailedSettlementRepo.Create(
		tc.Ctx,
		generated.CreateFailedSettlementParams{
			TradeID:      tradeID,
			BuyOrderID:   buyOrderID,
			SellOrderID:  sellOrderID,
			BuyerID:      buyerID,
			SellerID:     sellerID,
			Symbol:       symbol,
			Price:        50000,
			Quantity:     1,
			ErrorMessage: "test failure",
		},
	)
	require.NoError(t, err)

	// Bring retry_count to the maximum retry limit.
	for i := 0; i < 10; i++ {
		err := tc.FailedSettlementRepo.IncrementRetryCount(
			tc.Ctx,
			failed.ID,
		)
		require.NoError(t, err)
	}

	// Start worker.
	ctx, cancel := context.WithCancel(tc.Ctx)
	defer cancel()

	worker.Start(ctx)

	// Worker should detect retry_count >= 10 and dead-letter it.
	require.Eventually(t, func() bool {
		current, err := tc.FailedSettlementRepo.Get(
			tc.Ctx,
			failed.ID,
		)
		if err != nil {
			return false
		}

		return current.IsDead
	}, 2*time.Second, 10*time.Millisecond)

	current, err := tc.FailedSettlementRepo.Get(
		tc.Ctx,
		failed.ID,
	)
	require.NoError(t, err)

	require.True(t, current.IsDead)
	require.False(t, current.Resolved)
	require.Equal(t, int32(10), current.RetryCount)

	// Dead settlements must no longer appear in unresolved queue.
	unresolved, err := tc.FailedSettlementRepo.ListUnresolved(tc.Ctx)
	require.NoError(t, err)

	for _, item := range unresolved {
		require.NotEqual(t, failed.ID, item.ID)
	}

}

func TestFailedSettlementWorker_CrashWindowIdempotency(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := settlementservice.New(
		tc.TxManager,
		tc.UserDispatcher,
	)

	// ----------------------------------------------------------------------
	// Fixture
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

	base := time.Now().UnixNano()

	buyerID := base
	sellerID := base + 1
	buyOrderID := base + 2
	sellOrderID := base + 3
	tradeID := base + 4

	// ----------------------------------------------------------------------
	// Users
	// ----------------------------------------------------------------------

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

	req := settlementservice.SettlementRequest{
		TradeID:     tradeID,
		BuyOrderID:  buyOrderID,
		SellOrderID: sellOrderID,
		BuyerID:     buyerID,
		SellerID:    sellerID,
		Symbol:      symbol,
		BaseAsset:   "BTC",
		QuoteAsset:  "USDT",
		Price:       50000,
		Quantity:    1,
	}

	// ----------------------------------------------------------------------
	// Simulate crash window
	//
	// The failure record exists, but settlement itself has already
	// successfully committed. This represents a worker crash occurring
	// between Settle() and Resolve().
	// ----------------------------------------------------------------------

	failed, err := tc.FailedSettlementRepo.Create(
		tc.Ctx,
		generated.CreateFailedSettlementParams{
			TradeID:      tradeID,
			BuyOrderID:   buyOrderID,
			SellOrderID:  sellOrderID,
			BuyerID:      buyerID,
			SellerID:     sellerID,
			Symbol:       symbol,
			Price:        50000,
			Quantity:     1,
			ErrorMessage: "simulated crash after successful settlement",
		},
	)
	require.NoError(t, err)

	// Settlement succeeds before the worker gets a chance to retry.
	require.NoError(t, service.Settle(tc.Ctx, req))

	// ----------------------------------------------------------------------
	// Capture committed state
	// ----------------------------------------------------------------------

	buyerBTCBefore, err := tc.WalletRepo.Get(tc.Ctx, buyerID, "BTC")
	require.NoError(t, err)

	buyerUSDTBefore, err := tc.WalletRepo.Get(tc.Ctx, buyerID, "USDT")
	require.NoError(t, err)

	sellerBTCBefore, err := tc.WalletRepo.Get(tc.Ctx, sellerID, "BTC")
	require.NoError(t, err)

	sellerUSDTBefore, err := tc.WalletRepo.Get(tc.Ctx, sellerID, "USDT")
	require.NoError(t, err)

	buyOrderBefore, err := tc.OrderRepo.GetByID(tc.Ctx, buyOrderID)
	require.NoError(t, err)

	sellOrderBefore, err := tc.OrderRepo.GetByID(tc.Ctx, sellOrderID)
	require.NoError(t, err)

	// Confirm the trade really exists before the retry.
	trade, err := tc.TradeRepo.GetByID(tc.Ctx, tradeID)
	require.NoError(t, err)
	require.Equal(t, tradeID, trade.ID)

	// ----------------------------------------------------------------------
	// Start worker.
	// ----------------------------------------------------------------------

	ctx, cancel := context.WithCancel(tc.Ctx)
	defer cancel()

	w := worker.NewFailedSettlementWorker(
		service,
		tc.FailedSettlementRepo,
		tc.SymbolRepo,
		zap.NewNop(),
		10*time.Millisecond,
	)

	w.Start(ctx)

	// ----------------------------------------------------------------------
	// Worker should retry the same TradeID and resolve the failure.
	// ----------------------------------------------------------------------

	require.Eventually(t, func() bool {
		current, err := tc.FailedSettlementRepo.Get(
			tc.Ctx,
			failed.ID,
		)
		if err != nil {
			return false
		}

		return current.Resolved
	}, 2*time.Second, 10*time.Millisecond)

	// ----------------------------------------------------------------------
	// Failure record resolved.
	// ----------------------------------------------------------------------

	failedAfter, err := tc.FailedSettlementRepo.Get(
		tc.Ctx,
		failed.ID,
	)
	require.NoError(t, err)

	require.True(t, failedAfter.Resolved)
	require.False(t, failedAfter.IsDead)
	require.Equal(t, int32(0), failedAfter.RetryCount)

	// ----------------------------------------------------------------------
	// Trade must still exist exactly once.
	// ----------------------------------------------------------------------

	tradeAfter, err := tc.TradeRepo.GetByID(tc.Ctx, tradeID)
	require.NoError(t, err)
	require.Equal(t, tradeID, tradeAfter.ID)

	// ----------------------------------------------------------------------
	// Wallets must NOT be mutated a second time.
	// ----------------------------------------------------------------------

	buyerBTCAfter, err := tc.WalletRepo.Get(tc.Ctx, buyerID, "BTC")
	require.NoError(t, err)

	buyerUSDTAfter, err := tc.WalletRepo.Get(tc.Ctx, buyerID, "USDT")
	require.NoError(t, err)

	sellerBTCAfter, err := tc.WalletRepo.Get(tc.Ctx, sellerID, "BTC")
	require.NoError(t, err)

	sellerUSDTAfter, err := tc.WalletRepo.Get(tc.Ctx, sellerID, "USDT")
	require.NoError(t, err)

	require.Equal(t, buyerBTCBefore.Available, buyerBTCAfter.Available)
	require.Equal(t, buyerBTCBefore.Locked, buyerBTCAfter.Locked)

	require.Equal(t, buyerUSDTBefore.Available, buyerUSDTAfter.Available)
	require.Equal(t, buyerUSDTBefore.Locked, buyerUSDTAfter.Locked)

	require.Equal(t, sellerBTCBefore.Available, sellerBTCAfter.Available)
	require.Equal(t, sellerBTCBefore.Locked, sellerBTCAfter.Locked)

	require.Equal(t, sellerUSDTBefore.Available, sellerUSDTAfter.Available)
	require.Equal(t, sellerUSDTBefore.Locked, sellerUSDTAfter.Locked)

	// ----------------------------------------------------------------------
	// Orders must NOT be mutated a second time.
	// ----------------------------------------------------------------------

	buyOrderAfter, err := tc.OrderRepo.GetByID(tc.Ctx, buyOrderID)
	require.NoError(t, err)

	sellOrderAfter, err := tc.OrderRepo.GetByID(tc.Ctx, sellOrderID)
	require.NoError(t, err)

	require.Equal(t, buyOrderBefore.Filled, buyOrderAfter.Filled)
	require.Equal(t, buyOrderBefore.Remaining, buyOrderAfter.Remaining)

	require.Equal(t, sellOrderBefore.Filled, sellOrderAfter.Filled)
	require.Equal(t, sellOrderBefore.Remaining, sellOrderAfter.Remaining)
}
