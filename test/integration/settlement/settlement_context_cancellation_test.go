package settlement

// ---------------------------------------------------------------------------
// What this file is testing
//
// Service.Settle runs everything inside a single pgx transaction
// (tx.Manager.WithTransaction): claim the trade ID, load both orders,
// validate, debit the buyer's locked funds, credit the seller, update
// positions, update order fill state - then commit. If the underlying
// connection is dropped or reset partway through - a real possibility on
// any network hop between the settlement worker and Postgres - whatever
// statement was in flight fails, WithTransaction's deferred tx.Rollback
// runs, and Settle returns an error.
//
// What this test checks is not "does it return an error" - that's not
// interesting on its own - but "is the resulting database state ALWAYS
// either fully-settled or fully-untouched, never something in between."
// A trade row that exists with no matching wallet debit, or a debited
// wallet with no trade row, is exactly the kind of corruption that a
// dropped connection could cause in a hand-rolled or poorly-scoped
// transaction, and it's what this test exists to rule out for this one.
//
// There's no reliable way to make a real TCP connection drop land on one
// exact statement out of the several Settle() issues, and trying to would
// make this test flake by construction. Instead this sweeps a wide range
// of context timeouts - from far too short to succeed at all, up through
// comfortably long - so that across the whole run, cancellation lands at
// many different points inside the transaction body, including places
// that can't be predicted or named in advance. The invariant is checked
// after every single attempt, at every timeout.
// ---------------------------------------------------------------------------

import (
	"context"
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

// settlementTimeoutSweep spans "certainly too short to finish" through
// "certainly long enough to finish" so that, across the whole sweep,
// cancellation lands at many different points inside the transaction -
// including points too fine-grained to target by hand.
var settlementTimeoutSweep = []time.Duration{
	0,
	1 * time.Microsecond,
	5 * time.Microsecond,
	10 * time.Microsecond,
	25 * time.Microsecond,
	50 * time.Microsecond,
	100 * time.Microsecond,
	250 * time.Microsecond,
	500 * time.Microsecond,
	1 * time.Millisecond,
	2 * time.Millisecond,
	5 * time.Millisecond,
	10 * time.Millisecond,
	25 * time.Millisecond,
	50 * time.Millisecond,
	200 * time.Millisecond, // control: should always be long enough to fully commit
}

func TestChaos_SettlementInterruptedAtAnyPoint_IsAlwaysAllOrNothing(t *testing.T) {
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
		},
	)
	require.NoError(t, err)

	const price = int64(50000)
	const quantity = int64(1)

	sawFullCommit := false
	sawFullRollback := false

	for i, timeout := range settlementTimeoutSweep {

		// Fresh users, wallets, orders and trade ID every iteration -
		// each attempt must be judged independently, since the trade ID
		// is the idempotency key and a prior success would otherwise
		// make a later attempt in the same sweep a guaranteed no-op.
		base := time.Now().UnixNano() + int64(i)*1000
		buyerID := base
		sellerID := base + 1
		buyOrderID := base + 2
		sellOrderID := base + 3
		tradeID := base + 4

		_, err := tc.UserRepo.Create(tc.Ctx, generated.CreateUserParams{
			ID:        buyerID,
			Email:     uuid.NewString() + "@buyer.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
		require.NoError(t, err)

		_, err = tc.UserRepo.Create(tc.Ctx, generated.CreateUserParams{
			ID:        sellerID,
			Email:     uuid.NewString() + "@seller.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
		require.NoError(t, err)

		_, err = tc.WalletRepo.Create(tc.Ctx, generated.CreateWalletParams{
			UserID: buyerID, Asset: "USDT", Locked: price * quantity,
		})
		require.NoError(t, err)

		_, err = tc.WalletRepo.Create(tc.Ctx, generated.CreateWalletParams{
			UserID: buyerID, Asset: "BTC",
		})
		require.NoError(t, err)

		_, err = tc.WalletRepo.Create(tc.Ctx, generated.CreateWalletParams{
			UserID: sellerID, Asset: "BTC", Locked: quantity,
		})
		require.NoError(t, err)

		_, err = tc.WalletRepo.Create(tc.Ctx, generated.CreateWalletParams{
			UserID: sellerID, Asset: "USDT",
		})
		require.NoError(t, err)

		orderPrice := pgtype.Int8{Int64: price, Valid: true}

		_, err = tc.OrderRepo.Create(tc.Ctx, generated.CreateOrderParams{
			ID: buyOrderID, UserID: buyerID, Symbol: symbol,
			Side: "BUY", OrderType: "LIMIT", TimeInForce: "GTC",
			Status: string(constants.OrderStatusOpen),
			Price:  orderPrice, Quantity: quantity, Remaining: quantity,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		})
		require.NoError(t, err)

		_, err = tc.OrderRepo.Create(tc.Ctx, generated.CreateOrderParams{
			ID: sellOrderID, UserID: sellerID, Symbol: symbol,
			Side: "SELL", OrderType: "LIMIT", TimeInForce: "GTC",
			Status: string(constants.OrderStatusOpen),
			Price:  orderPrice, Quantity: quantity, Remaining: quantity,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		})
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
			Price:       price,
			Quantity:    quantity,
			ExecutedAt:  time.Now(),
		}

		// The injected failure: a context that may expire at any point
		// during the transaction - standing in for a connection that
		// drops, resets, or times out partway through, at a point we
		// don't get to choose.
		ctx, cancel := context.WithTimeout(tc.Ctx, timeout)
		settleErr := service.Settle(ctx, req)
		cancel()

		// ---------------------------------------------------------
		// Check the invariant: trade existence and wallet debit must
		// agree. Either both happened (full commit) or neither did
		// (full rollback) - never one without the other.
		// ---------------------------------------------------------

		_, tradeErr := tc.TradeRepo.GetByID(tc.Ctx, tradeID)
		tradeExists := tradeErr == nil

		buyerUSDT, walErr := tc.WalletRepo.Get(tc.Ctx, buyerID, "USDT")
		require.NoError(t, walErr, "wallet row must always exist regardless of settlement outcome")

		buyOrder, ordErr := tc.OrderRepo.GetByID(tc.Ctx, buyOrderID)
		require.NoError(t, ordErr, "order row must always exist regardless of settlement outcome")

		switch {

		case tradeExists && buyerUSDT.Locked == 0 && buyOrder.Remaining == 0:
			// Fully committed: trade recorded, funds consumed, order filled.
			sawFullCommit = true

		case !tradeExists && buyerUSDT.Locked == price*quantity && buyOrder.Remaining == quantity:
			// Fully rolled back: as if Settle was never called.
			sawFullRollback = true

		default:
			t.Fatalf(
				"BUG at timeout=%v (iteration %d): settlement left a "+
					"partial state - tradeExists=%v buyerLocked=%d "+
					"(want 0 or %d) buyOrderRemaining=%d (want 0 or %d) - "+
					"settleErr=%v. This is a partially-applied transaction: "+
					"exactly what a dropped connection must never produce.",
				timeout, i, tradeExists, buyerUSDT.Locked, price*quantity,
				buyOrder.Remaining, quantity, settleErr,
			)
		}
	}

	// Sanity check on the sweep itself: if every attempt landed in the
	// same bucket, the timeout range didn't actually exercise both
	// regimes and the test wasn't testing anything.
	require.True(t, sawFullRollback,
		"sanity check: no timeout in the sweep was short enough to ever "+
			"interrupt settlement - widen settlementTimeoutSweep")
	require.True(t, sawFullCommit,
		"sanity check: no timeout in the sweep was long enough to let "+
			"settlement complete - widen settlementTimeoutSweep")
}