package settlement

import (
	"testing"
	"time"

	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/service/settlementservice"
	"velocity/pkg/constants"
	"velocity/test/integration"

	testhelpers "velocity/test/helpers"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// TestSettlement_ConcurrentWalletMutation_DifferentOrdersSameWallet verifies
// that two concurrent settlements can safely mutate the same buyer wallet
// even when they operate on different BUY orders.
//
// This is intentionally different from
// TestSettlement_ConcurrentIdempotency_DistinctTradesSameOrder.
//
// That test uses the SAME BUY order, so the order-level FOR UPDATE lock
// serializes the two settlements.
//
// Here:
//   - BUY order #1 and BUY order #2 are different orders.
//   - Both BUY orders belong to the SAME buyer.
//   - Both settlements therefore mutate the SAME buyer wallets.
//   - Because the orders are different, order locking cannot serialize
//     the two transactions.
//
// The wallet rows themselves must therefore be concurrency-safe.
//
// Expected:
//
//	buyer USDT:
//	    initial locked = 100000
//	    trade 1 consumes = 50000
//	    trade 2 consumes = 50000
//	    final locked = 0
//
//	buyer BTC:
//	    trade 1 receives = 1
//	    trade 2 receives = 1
//	    final available = 2
func TestSettlement_ConcurrentWalletMutation_DifferentOrdersSameWallet(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := settlementservice.New(
		tc.TxManager,
		tc.UserDispatcher,
	)

	// ------------------------------------------------------------------
	// Fixture 1
	//
	// Buyer A:
	//   USDT locked = 100000
	//   BTC         = 0
	//
	// BUY order #1:
	//   quantity = 2 BTC
	//
	// Seller A:
	//   BTC locked = 1
	// ------------------------------------------------------------------

	fx1 := newSettlementFixture(
		t,
		tc,
		2,
		1,
		100000,
		1,
	)

	// ------------------------------------------------------------------
	// Fixture 2
	//
	// We only need its seller.
	//
	// Fixture 2 creates another buyer as well, but that buyer is NOT used.
	// Its seller gives us a second independent SELL order belonging to the
	// same symbol.
	// ------------------------------------------------------------------

	fx2 := newSettlementFixture(
		t,
		tc,
		1,
		1,
		0,
		1,
	)

	// ------------------------------------------------------------------
	// Create BUY order #2 manually.
	//
	// IMPORTANT:
	//
	// This order belongs to fx1.buyerID.
	//
	// Therefore:
	//
	//   BUY #1 -> fx1.buyerID
	//   BUY #2 -> fx1.buyerID
	//
	// Both settlements will mutate the SAME buyer wallets, while operating
	// on DIFFERENT orders.
	// ------------------------------------------------------------------

	secondBuyOrderID := testhelpers.NextID()

	price := pgtype.Int8{
		Int64:  50000,
		Valid: true,
	}

	_, err := tc.OrderRepo.Create(
		tc.Ctx,
		generated.CreateOrderParams{
			ID:          secondBuyOrderID,
			UserID:      fx1.buyerID,
			Symbol:      fx1.symbol,
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

	// ------------------------------------------------------------------
	// Trade 1
	//
	// BUY order #1 + SELL order #1
	// Buyer = fx1.buyerID
	// Seller = fx1.sellerID
	// ------------------------------------------------------------------

	trade1 := settlementservice.SettlementRequest{
		TradeID:     testhelpers.NextID(),
		BuyOrderID:  fx1.buyOrderID,
		SellOrderID: fx1.sellOrderID,

		BuyerID:  fx1.buyerID,
		SellerID: fx1.sellerID,

		Symbol: fx1.symbol,

		BaseAsset:  "BTC",
		QuoteAsset: "USDT",

		Price:    50000,
		Quantity: 1,
	}

	// ------------------------------------------------------------------
	// Trade 2
	//
	// DIFFERENT BUY order + DIFFERENT SELL order.
	//
	// The buyer is intentionally the SAME buyer as trade 1.
	// ------------------------------------------------------------------

	trade2 := settlementservice.SettlementRequest{
		TradeID:     testhelpers.NextID(),
		BuyOrderID:  secondBuyOrderID,
		SellOrderID: fx2.sellOrderID,

		BuyerID:  fx1.buyerID,
		SellerID: fx2.sellerID,

		Symbol: fx1.symbol,

		BaseAsset:  "BTC",
		QuoteAsset: "USDT",

		Price:    50000,
		Quantity: 1,
	}

	// ------------------------------------------------------------------
	// Execute both settlements concurrently.
	//
	// There are two independent orders, so an order-level FOR UPDATE lock
	// must NOT be sufficient to protect the shared buyer wallet.
	// ------------------------------------------------------------------

	errs := runConcurrently(2, func(i int) error {
		if i == 0 {
			return service.Settle(tc.Ctx, trade1)
		}

		return service.Settle(tc.Ctx, trade2)
	})

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])

	// ------------------------------------------------------------------
	// Verify the shared buyer BTC wallet.
	//
	// Each trade must credit 1 BTC.
	//
	// Correct result:
	//   BTC available = 2
	// ------------------------------------------------------------------

	buyerBTC, err := tc.WalletRepo.Get(
		tc.Ctx,
		fx1.buyerID,
		"BTC",
	)
	require.NoError(t, err)

	require.EqualValues(
		t,
		2,
		buyerBTC.Available,
		"shared buyer BTC wallet lost a concurrent update",
	)

	// ------------------------------------------------------------------
	// Verify the shared buyer USDT wallet.
	//
	// Initial locked balance = 100000.
	//
	// Trade 1 consumes 50000.
	// Trade 2 consumes 50000.
	//
	// Correct result:
	//   locked = 0
	// ------------------------------------------------------------------

	buyerUSDT, err := tc.WalletRepo.Get(
		tc.Ctx,
		fx1.buyerID,
		"USDT",
	)
	require.NoError(t, err)

	require.EqualValues(
		t,
		0,
		buyerUSDT.Locked,
		"shared buyer USDT wallet lost a concurrent update",
	)

	// ------------------------------------------------------------------
	// Verify both BUY orders were independently settled.
	//
	// BUY #1 originally had quantity 2 and receives one fill:
	//   Filled    = 1
	//   Remaining = 1
	//
	// BUY #2 originally had quantity 1 and receives one fill:
	//   Filled    = 1
	//   Remaining = 0
	// ------------------------------------------------------------------

	buyOrder1, err := tc.OrderRepo.GetByID(
		tc.Ctx,
		fx1.buyOrderID,
	)
	require.NoError(t, err)

	require.EqualValues(t, 1, buyOrder1.Filled)
	require.EqualValues(t, 1, buyOrder1.Remaining)

	buyOrder2, err := tc.OrderRepo.GetByID(
		tc.Ctx,
		secondBuyOrderID,
	)
	require.NoError(t, err)

	require.EqualValues(t, 1, buyOrder2.Filled)
	require.EqualValues(t, 0, buyOrder2.Remaining)

	// ------------------------------------------------------------------
	// Verify both SELL orders were settled.
	// ------------------------------------------------------------------

	sellOrder1, err := tc.OrderRepo.GetByID(
		tc.Ctx,
		fx1.sellOrderID,
	)
	require.NoError(t, err)

	require.EqualValues(t, 1, sellOrder1.Filled)
	require.EqualValues(t, 0, sellOrder1.Remaining)

	sellOrder2, err := tc.OrderRepo.GetByID(
		tc.Ctx,
		fx2.sellOrderID,
	)
	require.NoError(t, err)

	require.EqualValues(t, 1, sellOrder2.Filled)
	require.EqualValues(t, 0, sellOrder2.Remaining)
}