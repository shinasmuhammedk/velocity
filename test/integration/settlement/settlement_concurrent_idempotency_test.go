package settlement

import (
	"sync"
	"testing"

	"velocity/internal/service/settlementservice"
	"velocity/internal/userstream"
	"velocity/test/integration"

	testhelpers "velocity/test/helpers"

	"github.com/stretchr/testify/require"
)

// runConcurrently starts n workers, blocks them all on a shared gate so
// they fire as close to simultaneously as possible, then waits for every
// worker to finish. It returns each worker's error in call order.
func runConcurrently(n int, work func(i int) error) []error {
	var (
		wg   sync.WaitGroup
		gate = make(chan struct{})
		errs = make([]error, n)
	)

	for i := 0; i < n; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()
			<-gate
			errs[i] = work(i)
		}(i)
	}

	close(gate)
	wg.Wait()

	return errs
}

// TestSettlement_ConcurrentIdempotency_SameTradeID_AppliedExactlyOnce fires
// the *identical* settlement request from many goroutines at once,
// simulating a duplicate delivery / at-least-once retry racing itself.
// The trade-ID uniqueness constraint (INSERT ... ON CONFLICT DO NOTHING
// RETURNING) is what's supposed to make this safe: exactly one caller
// should observe a real settlement, and every other caller should see
// the no-op "already settled" path - with no partial or doubled effects
// anywhere, including in the events dispatched to users.
func TestSettlement_ConcurrentIdempotency_SameTradeID_AppliedExactlyOnce(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := settlementservice.New(tc.TxManager, tc.UserDispatcher)

	fx := newSettlementFixture(t, tc, 1, 1, 50000, 1)

	req := settlementservice.SettlementRequest{
		TradeID:     testhelpers.NextID(),
		BuyOrderID:  fx.buyOrderID,
		SellOrderID: fx.sellOrderID,

		BuyerID:  fx.buyerID,
		SellerID: fx.sellerID,

		Symbol: fx.symbol,

		BaseAsset:  "BTC",
		QuoteAsset: "USDT",

		Price:    50000,
		Quantity: 1,
	}

	const workers = 20

	errs := runConcurrently(workers, func(int) error {
		return service.Settle(tc.Ctx, req)
	})

	for i, err := range errs {
		require.NoErrorf(t, err, "worker %d returned an error", i)
	}

	// ------------------------------------------------------------------
	// Exactly one settlement's worth of state, no matter how many
	// goroutines raced to apply it.
	// ------------------------------------------------------------------

	buyerBTC, err := tc.WalletRepo.Get(tc.Ctx, fx.buyerID, "BTC")
	require.NoError(t, err)
	require.EqualValues(t, 1, buyerBTC.Available)

	buyerUSDT, err := tc.WalletRepo.Get(tc.Ctx, fx.buyerID, "USDT")
	require.NoError(t, err)
	require.EqualValues(t, 0, buyerUSDT.Locked)

	sellerBTC, err := tc.WalletRepo.Get(tc.Ctx, fx.sellerID, "BTC")
	require.NoError(t, err)
	require.EqualValues(t, 0, sellerBTC.Locked)

	sellerUSDT, err := tc.WalletRepo.Get(tc.Ctx, fx.sellerID, "USDT")
	require.NoError(t, err)
	require.EqualValues(t, 50000, sellerUSDT.Available)

	buyOrder, err := tc.OrderRepo.GetByID(tc.Ctx, fx.buyOrderID)
	require.NoError(t, err)
	require.EqualValues(t, 1, buyOrder.Filled)
	require.EqualValues(t, 0, buyOrder.Remaining)

	sellOrder, err := tc.OrderRepo.GetByID(tc.Ctx, fx.sellOrderID)
	require.NoError(t, err)
	require.EqualValues(t, 1, sellOrder.Filled)
	require.EqualValues(t, 0, sellOrder.Remaining)

	// ------------------------------------------------------------------
	// Exactly one settlement's worth of *events*, too. If the trade-ID
	// guard only protected the database and not the event dispatch that
	// follows it, this is where a double-broadcast would show up.
	// ------------------------------------------------------------------

	require.Len(t, fx.buyerSub.MessagesOfType(userstream.EventBalanceUpdated), 2)
	require.Len(t, fx.sellerSub.MessagesOfType(userstream.EventBalanceUpdated), 2)

	require.Len(t, fx.buyerSub.MessagesOfType(userstream.EventTradeExecuted), 1)
	require.Len(t, fx.sellerSub.MessagesOfType(userstream.EventTradeExecuted), 1)

	require.Len(t, fx.buyerSub.MessagesOfType(userstream.EventPositionUpdated), 1)
	require.Len(t, fx.sellerSub.MessagesOfType(userstream.EventPositionUpdated), 1)

	require.Len(t, fx.buyerSub.MessagesOfType(userstream.EventOrderFilled), 1)
	require.Len(t, fx.sellerSub.MessagesOfType(userstream.EventOrderFilled), 1)
}

// TestSettlement_ConcurrentIdempotency_DistinctTradesSameOrder_NoLostUpdate
// settles TWO DIFFERENT trades against the SAME resting buy order at the
// same time - e.g. a large resting order filled by two smaller incoming
// orders, picked up by two settlement workers concurrently. Unlike the
// same-trade-ID case above, there is no unique-constraint guard here:
// each settlement reads the buy order's current remaining/filled with a
// plain SELECT and writes back absolute values with a plain UPDATE. If
// that read-modify-write isn't isolated, one trade's update can silently
// overwrite the other's, permanently under-reporting how much of the
// order has actually filled.
//
// If this test fails, it is exposing a real gap: internal/persistence/
// postgres/queries/orders.sql's GetOrderByID has no FOR UPDATE, and
// UpdateOrderAfterTrade writes absolute (not relative) columns. The fix
// is to lock the order row for the duration of the settlement
// transaction (a dedicated "GetOrderByIDForUpdate ... FOR UPDATE" query
// used only inside Settle, since the plain GetOrderByID is also used
// outside of transactions elsewhere) or to rewrite the update as an
// atomic relative decrement guarded by a WHERE remaining >= $qty clause,
// the same pattern positions.sql already uses safely.
func TestSettlement_ConcurrentIdempotency_DistinctTradesSameOrder_NoLostUpdate(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := settlementservice.New(tc.TxManager, tc.UserDispatcher)

	// One buy order for 2, filled by two different sell orders for 1
	// each, so both trades touch the same buy order's row concurrently.
	fx := newSettlementFixture(t, tc, 2, 1, 100000, 1)

	// A second, independent seller + sell order for the second trade.
	fx2 := newSettlementFixture(t, tc, 0, 1, 0, 1)

	trade1 := settlementservice.SettlementRequest{
		TradeID:     testhelpers.NextID(),
		BuyOrderID:  fx.buyOrderID,
		SellOrderID: fx.sellOrderID,

		BuyerID:  fx.buyerID,
		SellerID: fx.sellerID,

		Symbol: fx.symbol,

		BaseAsset:  "BTC",
		QuoteAsset: "USDT",

		Price:    50000,
		Quantity: 1,
	}

	trade2 := settlementservice.SettlementRequest{
		TradeID:     testhelpers.NextID(),
		BuyOrderID:  fx.buyOrderID, // same resting buy order as trade1
		SellOrderID: fx2.sellOrderID,

		BuyerID:  fx.buyerID,
		SellerID: fx2.sellerID,

		Symbol: fx.symbol,

		BaseAsset:  "BTC",
		QuoteAsset: "USDT",

		Price:    50000,
		Quantity: 1,
	}

	requests := []settlementservice.SettlementRequest{trade1, trade2}

	errs := runConcurrently(len(requests), func(i int) error {
		return service.Settle(tc.Ctx, requests[i])
	})

	for i, err := range errs {
		require.NoErrorf(t, err, "trade %d settlement returned an error", i)
	}

	// ------------------------------------------------------------------
	// The buy order took two fills of 1 each: it must end up fully
	// filled, not half-filled with one update lost to the other.
	// ------------------------------------------------------------------

	buyOrder, err := tc.OrderRepo.GetByID(tc.Ctx, fx.buyOrderID)
	require.NoError(t, err)
	require.EqualValuesf(t, 2, buyOrder.Filled,
		"lost update: buy order Filled=%d, want 2 (one trade's UpdateOrderAfterTrade overwrote the other's)",
		buyOrder.Filled)
	require.EqualValuesf(t, 0, buyOrder.Remaining,
		"lost update: buy order Remaining=%d, want 0", buyOrder.Remaining)

	// ------------------------------------------------------------------
	// Each sell order settled independently and should be unaffected by
	// the race on the shared buy order.
	// ------------------------------------------------------------------

	sellOrder1, err := tc.OrderRepo.GetByID(tc.Ctx, fx.sellOrderID)
	require.NoError(t, err)
	require.EqualValues(t, 1, sellOrder1.Filled)
	require.EqualValues(t, 0, sellOrder1.Remaining)

	sellOrder2, err := tc.OrderRepo.GetByID(tc.Ctx, fx2.sellOrderID)
	require.NoError(t, err)
	require.EqualValues(t, 1, sellOrder2.Filled)
	require.EqualValues(t, 0, sellOrder2.Remaining)

	// ------------------------------------------------------------------
	// The buyer's wallets are also shared, unlocked state touched by
	// both trades: same lost-update risk, this time on wallets.sql's
	// GetWallet/UpdateWallet.
	// ------------------------------------------------------------------

	buyerBTC, err := tc.WalletRepo.Get(tc.Ctx, fx.buyerID, "BTC")
	require.NoError(t, err)
	require.EqualValuesf(t, 2, buyerBTC.Available,
		"lost update: buyer BTC available=%d, want 2 (one trade's wallet UPDATE overwrote the other's)",
		buyerBTC.Available)

	buyerUSDT, err := tc.WalletRepo.Get(tc.Ctx, fx.buyerID, "USDT")
	require.NoError(t, err)
	require.EqualValuesf(t, 0, buyerUSDT.Locked,
		"lost update: buyer USDT locked=%d, want 0", buyerUSDT.Locked)
}