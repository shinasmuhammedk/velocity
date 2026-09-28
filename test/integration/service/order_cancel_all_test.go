package integration

import (
	"testing"
	"time"

	"velocity/internal/domain/order"
	"velocity/internal/engine/registry"
	"velocity/internal/engine/snapshot"
	"velocity/internal/engine/wal"
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/service/orderservice"
	"velocity/pkg/constants"
	"velocity/pkg/errors"
	"velocity/test/integration"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// newCancelAllTestService builds an orderservice.Service with a real
// (in-memory) matching-engine registry but nil risk/wallet/dispatcher/
// id-generator dependencies. CancelAll and the Cancel path it drives
// only touch orderRepo, symbolRepo, userRepo, the registry, and (on a
// successful cancel) the UserDispatcher - so this is safe as long as
// tests here don't exercise the happy-path notification.
func newCancelAllTestService(t *testing.T, tc *integration.TestContext) (*orderservice.Service, *registry.Registry) {
	t.Helper()

	walManager := wal.NewManager(
		t.TempDir(),
		wal.NewJSONSerializer(),
	)

	reg := registry.New(
		&snapshot.MockWriter{},
		walManager,
	)
	t.Cleanup(func() {
		_ = reg.Shutdown()
	})

	svc := orderservice.New(
		tc.OrderRepo,
		tc.SymbolRepo,
		tc.UserRepo,
		tc.TradeRepo,
		nil, // risk
		nil, // wallet
		reg,
		nil, // logger
		tc.UserDispatcher,
		nil, // idGenerator
	)

	return svc, reg
}

func createCancelAllTestUser(t *testing.T, tc *integration.TestContext) int64 {
	t.Helper()

	userID := time.Now().UnixNano()

	_, err := tc.UserRepo.Create(
		tc.Ctx,
		generated.CreateUserParams{
			ID:        userID,
			Email:     uuid.NewString() + "@test.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)

	return userID
}

func createCancelAllTestSymbol(t *testing.T, tc *integration.TestContext) string {
	t.Helper()

	symbol := "SYM_" + uuid.NewString()[:8]

	_, err := tc.SymbolRepo.Create(
		tc.Ctx,
		generated.CreateSymbolParams{
			Symbol:      symbol,
			DisplayName: symbol,
			TickSize:    1,
			LotSize:     1,
			IsActive:    true,
		},
	)
	require.NoError(t, err)

	return symbol
}

func createCancelAllTestOrder(
	t *testing.T,
	tc *integration.TestContext,
	orderID int64,
	userID int64,
	symbol string,
	status string,
) {
	t.Helper()

	_, err := tc.OrderRepo.Create(
		tc.Ctx,
		generated.CreateOrderParams{
			ID:          orderID,
			UserID:      userID,
			Symbol:      symbol,
			Side:        "BUY",
			OrderType:   "LIMIT",
			TimeInForce: "GTC",
			Status:      status,

			Price: pgtype.Int8{
				Int64: 100,
				Valid: true,
			},

			StopPrice: 0,
			Quantity:  10,
			Remaining: 10,
			Filled:    0,

			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	)
	require.NoError(t, err)
}

func TestCancelAll_UnknownUser(t *testing.T) {
	tc := integration.NewTestContext(t)
	svc, _ := newCancelAllTestService(t, tc)

	cancelled, err := svc.CancelAll(
		tc.Ctx,
		time.Now().UnixNano(), // never created
		"",
	)

	require.ErrorIs(t, err, errors.ErrUserNotFound)
	require.Equal(t, 0, cancelled)
}

func TestCancelAll_UnknownSymbol(t *testing.T) {
	tc := integration.NewTestContext(t)
	svc, _ := newCancelAllTestService(t, tc)

	userID := createCancelAllTestUser(t, tc)

	cancelled, err := svc.CancelAll(
		tc.Ctx,
		userID,
		"DOES_NOT_EXIST",
	)

	require.ErrorIs(t, err, errors.ErrSymbolNotFound)
	require.Equal(t, 0, cancelled)
}

func TestCancelAll_NoOpenOrders(t *testing.T) {
	tc := integration.NewTestContext(t)
	svc, _ := newCancelAllTestService(t, tc)

	userID := createCancelAllTestUser(t, tc)

	cancelled, err := svc.CancelAll(
		tc.Ctx,
		userID,
		"",
	)

	require.NoError(t, err)
	require.Equal(t, 0, cancelled)
}

// TestCancelAll_StopsOnUnexpectedError exercises the "hard fail" branch
// of the CancelAll loop: an order that is genuinely cancelable in the DB
// (status OPEN) but whose symbol has no live matching engine registered
// causes Cancel to return ErrEngineUnavailable, which is not one of the
// races CancelAll tolerates, so it must stop and surface the error
// rather than silently skipping the order.
func TestCancelAll_StopsOnUnexpectedError(t *testing.T) {
	tc := integration.NewTestContext(t)
	svc, _ := newCancelAllTestService(t, tc)

	userID := createCancelAllTestUser(t, tc)
	symbol := createCancelAllTestSymbol(t, tc)

	orderID := time.Now().UnixNano()
	createCancelAllTestOrder(t, tc, orderID, userID, symbol, "OPEN")

	// Deliberately do not call registry.Get(symbol) - no engine exists
	// for this symbol, so registry.Find inside Cancel will fail.

	cancelled, err := svc.CancelAll(
		tc.Ctx,
		userID,
		"",
	)

	require.ErrorIs(t, err, errors.ErrEngineUnavailable)
	require.Equal(t, 0, cancelled)

	// The order must be untouched - CancelAll must not mark it
	// cancelled in the DB when the engine call never succeeded.
	dbOrder, err := tc.OrderRepo.GetByID(tc.Ctx, orderID)
	require.NoError(t, err)
	require.Equal(t, "OPEN", dbOrder.Status)
}

// TestCancelAll_SkipsOrderMissingFromBook exercises the "tolerated race"
// branch: the order is cancelable per the DB (status OPEN) and its
// symbol does have a live engine, but the order was never actually
// placed into that engine's book. The engine reports ErrOrderNotFound,
// which CancelAll treats as an expected race (the order's state moved
// on between the list query and the cancel call) and skips rather than
// failing the whole batch.
func TestCancelAll_SkipsOrderMissingFromBook(t *testing.T) {
	tc := integration.NewTestContext(t)
	svc, reg := newCancelAllTestService(t, tc)

	userID := createCancelAllTestUser(t, tc)
	symbol := createCancelAllTestSymbol(t, tc)

	// Registers a live engine for the symbol, but the order below is
	// never added to its book.
	reg.Get(symbol)

	orderID := time.Now().UnixNano()
	createCancelAllTestOrder(t, tc, orderID, userID, symbol, "OPEN")

	cancelled, err := svc.CancelAll(
		tc.Ctx,
		userID,
		"",
	)

	require.NoError(t, err)
	require.Equal(t, 0, cancelled)
}

// TestCancelAll_CancelsOrderInBook is the happy path: an order that
// exists both in the DB (as OPEN) and in the matching engine's live
// book gets cancelled, counted, and its DB status updated.
func TestCancelAll_CancelsOrderInBook(t *testing.T) {
	tc := integration.NewTestContext(t)
	svc, reg := newCancelAllTestService(t, tc)

	userID := createCancelAllTestUser(t, tc)
	symbol := createCancelAllTestSymbol(t, tc)

	orderID := time.Now().UnixNano()
	createCancelAllTestOrder(t, tc, orderID, userID, symbol, "OPEN")

	eng := reg.Get(symbol)
	eng.OrderBook().AddOrder(&order.Order{
		ID:        orderID,
		UserID:    userID,
		Symbol:    symbol,
		Side:      constants.OrderSideBuy,
		Type:      constants.OrderTypeLimit,
		Status:    constants.OrderStatusOpen,
		Price:     100,
		Quantity:  10,
		Remaining: 10,
	})

	cancelled, err := svc.CancelAll(
		tc.Ctx,
		userID,
		symbol,
	)

	require.NoError(t, err)
	require.Equal(t, 1, cancelled)

	dbOrder, err := tc.OrderRepo.GetByID(tc.Ctx, orderID)
	require.NoError(t, err)
	require.Equal(t, "CANCELLED", dbOrder.Status)

	require.Nil(t, eng.OrderBook().GetOrder(orderID))
}
