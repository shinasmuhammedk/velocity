package settlement

import (
	"testing"
	"time"

	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/service/settlementservice"
	"velocity/internal/userstream"
	"velocity/pkg/constants"
	apperrors "velocity/pkg/errors"
	"velocity/test/integration"

	testhelpers "velocity/test/helpers"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// settlementFixture is shared scaffolding (symbol, users, wallets, orders)
// for the balance-event tests below. It mirrors the setup used by the
// other settlement integration tests, but also wires each user up to a
// FakeSubscriber so the test can inspect exactly what the settlement
// pipeline broadcasts.
type settlementFixture struct {
	symbol string

	buyerID  int64
	sellerID int64

	buyOrderID  int64
	sellOrderID int64

	buyerSub  *testhelpers.FakeSubscriber
	sellerSub *testhelpers.FakeSubscriber
}

func newSettlementFixture(
	t *testing.T,
	tc *integration.TestContext,
	buyQty, sellQty, buyerLockedQuote, sellerLockedBase int64,
) settlementFixture {
	t.Helper()

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

	buyerID := testhelpers.NextID()
	sellerID := testhelpers.NextID()
	buyOrderID := testhelpers.NextID()
	sellOrderID := testhelpers.NextID()

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
			UserID: buyerID,
			Asset:  "USDT",
			Locked: buyerLockedQuote,
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
			Locked: sellerLockedBase,
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

	price := pgtype.Int8{Int64: 50000, Valid: true}

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
			Quantity:    buyQty,
			Remaining:   buyQty,
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
			Quantity:    sellQty,
			Remaining:   sellQty,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	)
	require.NoError(t, err)

	buyerSub := testhelpers.NewFakeSubscriber()
	sellerSub := testhelpers.NewFakeSubscriber()

	tc.Hub.Subscribe(buyerID, buyerSub)
	tc.Hub.Subscribe(sellerID, sellerSub)

	t.Cleanup(func() {
		tc.Hub.Unsubscribe(buyerID, buyerSub)
		tc.Hub.Unsubscribe(sellerID, sellerSub)
	})

	return settlementFixture{
		symbol:      symbol,
		buyerID:     buyerID,
		sellerID:    sellerID,
		buyOrderID:  buyOrderID,
		sellOrderID: sellOrderID,
		buyerSub:    buyerSub,
		sellerSub:   sellerSub,
	}
}

// balanceUpdateFor extracts the BalanceUpdate payload of the given
// balance.updated message, failing the test if the message is missing
// or malformed.
func balanceUpdateFor(t *testing.T, msg userstream.Message) userstream.BalanceUpdate {
	t.Helper()

	require.Equal(t, string(userstream.EventBalanceUpdated), msg.Type)

	update, ok := msg.Data.(userstream.BalanceUpdate)
	require.True(t, ok, "balance.updated message payload was not a userstream.BalanceUpdate")

	return update
}

func TestSettlement_BalanceEvents_Success(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := settlementservice.New(tc.TxManager, tc.UserDispatcher)

	fx := newSettlementFixture(t, tc, 1, 1, 50000, 1)

	tradeID := testhelpers.NextID()

	err := service.Settle(
		tc.Ctx,
		settlementservice.SettlementRequest{
			TradeID:     tradeID,
			BuyOrderID:  fx.buyOrderID,
			SellOrderID: fx.sellOrderID,

			BuyerID:  fx.buyerID,
			SellerID: fx.sellerID,

			Symbol: fx.symbol,

			BaseAsset:  "BTC",
			QuoteAsset: "USDT",

			Price:    50000,
			Quantity: 1,
		},
	)
	require.NoError(t, err)

	// ------------------------------------------------------------------
	// Each side should receive exactly one balance.updated event per
	// asset it holds in this trade (base + quote) - no more, no less.
	// ------------------------------------------------------------------

	buyerBalanceMsgs := fx.buyerSub.MessagesOfType(userstream.EventBalanceUpdated)
	sellerBalanceMsgs := fx.sellerSub.MessagesOfType(userstream.EventBalanceUpdated)

	require.Len(t, buyerBalanceMsgs, 2)
	require.Len(t, sellerBalanceMsgs, 2)

	// ------------------------------------------------------------------
	// The events must reflect the post-settlement wallet state exactly -
	// not a stale, pre-settlement, or partially-applied snapshot.
	// ------------------------------------------------------------------

	buyerBTC, err := tc.WalletRepo.Get(tc.Ctx, fx.buyerID, "BTC")
	require.NoError(t, err)

	buyerUSDT, err := tc.WalletRepo.Get(tc.Ctx, fx.buyerID, "USDT")
	require.NoError(t, err)

	sellerBTC, err := tc.WalletRepo.Get(tc.Ctx, fx.sellerID, "BTC")
	require.NoError(t, err)

	sellerUSDT, err := tc.WalletRepo.Get(tc.Ctx, fx.sellerID, "USDT")
	require.NoError(t, err)

	var buyerBaseSeen, buyerQuoteSeen bool

	for _, msg := range buyerBalanceMsgs {
		update := balanceUpdateFor(t, msg)

		switch update.Asset {
		case "BTC":
			buyerBaseSeen = true
			require.EqualValues(t, buyerBTC.Available, update.Available)
			require.EqualValues(t, buyerBTC.Locked, update.Locked)
		case "USDT":
			buyerQuoteSeen = true
			require.EqualValues(t, buyerUSDT.Available, update.Available)
			require.EqualValues(t, buyerUSDT.Locked, update.Locked)
		default:
			t.Fatalf("unexpected asset in buyer balance event: %s", update.Asset)
		}
	}

	require.True(t, buyerBaseSeen, "buyer never received a BTC balance.updated event")
	require.True(t, buyerQuoteSeen, "buyer never received a USDT balance.updated event")

	var sellerBaseSeen, sellerQuoteSeen bool

	for _, msg := range sellerBalanceMsgs {
		update := balanceUpdateFor(t, msg)

		switch update.Asset {
		case "BTC":
			sellerBaseSeen = true
			require.EqualValues(t, sellerBTC.Available, update.Available)
			require.EqualValues(t, sellerBTC.Locked, update.Locked)
		case "USDT":
			sellerQuoteSeen = true
			require.EqualValues(t, sellerUSDT.Available, update.Available)
			require.EqualValues(t, sellerUSDT.Locked, update.Locked)
		default:
			t.Fatalf("unexpected asset in seller balance event: %s", update.Asset)
		}
	}

	require.True(t, sellerBaseSeen, "seller never received a BTC balance.updated event")
	require.True(t, sellerQuoteSeen, "seller never received a USDT balance.updated event")

	// ------------------------------------------------------------------
	// Sibling events for a full fill on both sides.
	// ------------------------------------------------------------------

	require.Len(t, fx.buyerSub.MessagesOfType(userstream.EventTradeExecuted), 1)
	require.Len(t, fx.sellerSub.MessagesOfType(userstream.EventTradeExecuted), 1)

	require.Len(t, fx.buyerSub.MessagesOfType(userstream.EventPositionUpdated), 1)
	require.Len(t, fx.sellerSub.MessagesOfType(userstream.EventPositionUpdated), 1)

	require.Len(t, fx.buyerSub.MessagesOfType(userstream.EventOrderFilled), 1)
	require.Len(t, fx.sellerSub.MessagesOfType(userstream.EventOrderFilled), 1)

	require.Len(t, fx.buyerSub.MessagesOfType(userstream.EventOrderPartiallyFilled), 0)
	require.Len(t, fx.sellerSub.MessagesOfType(userstream.EventOrderPartiallyFilled), 0)
}

func TestSettlement_BalanceEvents_PartialFill(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := settlementservice.New(tc.TxManager, tc.UserDispatcher)

	// Buyer wants 2, seller only offers 1: the trade settles 1, fully
	// filling the sell order while leaving the buy order partially
	// filled.
	fx := newSettlementFixture(t, tc, 2, 1, 100000, 1)

	tradeID := testhelpers.NextID()

	err := service.Settle(
		tc.Ctx,
		settlementservice.SettlementRequest{
			TradeID:     tradeID,
			BuyOrderID:  fx.buyOrderID,
			SellOrderID: fx.sellOrderID,

			BuyerID:  fx.buyerID,
			SellerID: fx.sellerID,

			Symbol: fx.symbol,

			BaseAsset:  "BTC",
			QuoteAsset: "USDT",

			Price:    50000,
			Quantity: 1,
		},
	)
	require.NoError(t, err)

	// Balance events fire the same way regardless of which side was
	// fully or partially filled - both users still moved both assets.
	require.Len(t, fx.buyerSub.MessagesOfType(userstream.EventBalanceUpdated), 2)
	require.Len(t, fx.sellerSub.MessagesOfType(userstream.EventBalanceUpdated), 2)

	// But the order-status event dispatched to each side must reflect
	// its own remaining quantity, not the counterparty's.
	require.Len(t, fx.buyerSub.MessagesOfType(userstream.EventOrderPartiallyFilled), 1)
	require.Len(t, fx.buyerSub.MessagesOfType(userstream.EventOrderFilled), 0)

	require.Len(t, fx.sellerSub.MessagesOfType(userstream.EventOrderFilled), 1)
	require.Len(t, fx.sellerSub.MessagesOfType(userstream.EventOrderPartiallyFilled), 0)
}

func TestSettlement_BalanceEvents_DuplicateTrade_NoExtraEvents(t *testing.T) {
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

	require.NoError(t, service.Settle(tc.Ctx, req))

	buyerCountAfterFirst := fx.buyerSub.Count()
	sellerCountAfterFirst := fx.sellerSub.Count()

	require.Greater(t, buyerCountAfterFirst, 0)
	require.Greater(t, sellerCountAfterFirst, 0)

	// Re-settling the exact same trade ID must be a pure no-op: no
	// further state mutation, and therefore no further events -
	// including balance events - since Settle only dispatches after a
	// real commit, and the duplicate path commits nothing.
	require.NoError(t, service.Settle(tc.Ctx, req))

	require.Equal(t, buyerCountAfterFirst, fx.buyerSub.Count())
	require.Equal(t, sellerCountAfterFirst, fx.sellerSub.Count())
}

func TestSettlement_BalanceEvents_FailedSettlement_NoEvents(t *testing.T) {
	tc := integration.NewTestContext(t)

	service := settlementservice.New(tc.TxManager, tc.UserDispatcher)

	// Seller only has 1 BTC locked; requesting a settlement quantity
	// larger than either order's remaining quantity forces the
	// transaction to fail (and roll back) before any wallet, order, or
	// position row is touched.
	fx := newSettlementFixture(t, tc, 1, 1, 50000, 1)

	err := service.Settle(
		tc.Ctx,
		settlementservice.SettlementRequest{
			TradeID:     testhelpers.NextID(),
			BuyOrderID:  fx.buyOrderID,
			SellOrderID: fx.sellOrderID,

			BuyerID:  fx.buyerID,
			SellerID: fx.sellerID,

			Symbol: fx.symbol,

			BaseAsset:  "BTC",
			QuoteAsset: "USDT",

			Price:    50000,
			Quantity: 5, // exceeds both orders' remaining quantity of 1
		},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, apperrors.ErrInvalidQuantity)

	// A failed settlement must be invisible to both users: no partial
	// balance update, no stray trade or position event, nothing.
	require.Zero(t, fx.buyerSub.Count(), "buyer received events from a settlement that never committed")
	require.Zero(t, fx.sellerSub.Count(), "seller received events from a settlement that never committed")

	// And the wallets themselves must be untouched.
	buyerUSDT, err := tc.WalletRepo.Get(tc.Ctx, fx.buyerID, "USDT")
	require.NoError(t, err)
	require.EqualValues(t, 50000, buyerUSDT.Locked)

	sellerBTC, err := tc.WalletRepo.Get(tc.Ctx, fx.sellerID, "BTC")
	require.NoError(t, err)
	require.EqualValues(t, 1, sellerBTC.Locked)
}
