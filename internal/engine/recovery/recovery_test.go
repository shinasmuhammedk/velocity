package recovery_test

import (
	"context"
	"testing"
	"time"

	"velocity/internal/domain/order"
	"velocity/internal/engine/recovery"
	"velocity/internal/engine/registry"
	"velocity/internal/engine/snapshot"
	"velocity/internal/engine/wal"
	"velocity/internal/persistence/postgres/generated"
	"velocity/pkg/constants"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockOrderRepository struct {
	orders []generated.Order
}

func (m *mockOrderRepository) RecoveryOrders(
	ctx context.Context,
) ([]generated.Order, error) {
	return m.orders, nil
}

func TestRecoverySkipsSnapshotRestoredSymbols(
	t *testing.T,
) {
	serializer := snapshot.NewJSONSerializer()

	writer := snapshot.NewWriter(
		t.TempDir(),
		serializer,
	)

	walManager := wal.NewManager(
		t.TempDir(),
		wal.NewJSONSerializer(),
	)

	reg := registry.New(
		writer,
		walManager,
	)

	defer reg.Shutdown()

	repo := &mockOrderRepository{
		orders: []generated.Order{
			{
				ID:          1,
				UserID:      101,
				Symbol:      "BTCUSDT",
				Side:        "BUY",
				OrderType:   "LIMIT",
				TimeInForce: "GTC",
				Status:      "OPEN",

				Price: pgtype.Int8{
					Int64: 100000,
					Valid: true,
				},

				Quantity:  10,
				Remaining: 10,
				Filled:    0,

				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	r := recovery.New(
		repo,
		reg,
		zap.NewNop(),
	)

	alreadyRestored := map[string]bool{
		"BTCUSDT": true,
	}

	err := r.Load(
		context.Background(),
		alreadyRestored,
	)

	assert.NoError(
		t,
		err,
	)

	engine := reg.Get("BTCUSDT")

	assert.Equal(
		t,
		0,
		len(engine.OrderBook().Orders),
	)
}

func TestRecoveryLoadsOrdersForSymbolsWithoutSnapshots(
	t *testing.T,
) {
	serializer := snapshot.NewJSONSerializer()

	writer := snapshot.NewWriter(
		t.TempDir(),
		serializer,
	)

	walManager := wal.NewManager(
		t.TempDir(),
		wal.NewJSONSerializer(),
	)

	reg := registry.New(
		writer,
		walManager,
	)

	defer reg.Shutdown()

	repo := &mockOrderRepository{
		orders: []generated.Order{
			{
				ID:          2,
				UserID:      102,
				Symbol:      "ETHUSDT",
				Side:        "BUY",
				OrderType:   "LIMIT",
				TimeInForce: "GTC",
				Status:      "OPEN",

				Price: pgtype.Int8{
					Int64: 3000,
					Valid: true,
				},

				Quantity:  5,
				Remaining: 5,
				Filled:    0,

				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	r := recovery.New(
		repo,
		reg,
		zap.NewNop(),
	)

	err := r.Load(
		context.Background(),
		map[string]bool{},
	)

	assert.NoError(
		t,
		err,
	)

	engine := reg.Get("ETHUSDT")

	assert.Equal(
		t,
		1,
		len(engine.OrderBook().Orders),
	)
}

func (m *mockOrderRepository) Create(
	context.Context,
	generated.CreateOrderParams,
) (generated.Order, error) {
	panic("not implemented")
}

func (m *mockOrderRepository) GetByID(
	context.Context,
	int64,
) (generated.Order, error) {
	panic("not implemented")
}

func (m *mockOrderRepository) UpdateStatus(
	context.Context,
	generated.UpdateOrderStatusParams,
) error {
	panic("not implemented")
}

func (m *mockOrderRepository) ListByUser(
	context.Context,
	int64,
) ([]generated.Order, error) {
	panic("not implemented")
}

func (m *mockOrderRepository) ListOpenOrders(
	context.Context,
	string,
) ([]generated.Order, error) {
	panic("not implemented")
}

func (m *mockOrderRepository) GetPendingStopOrders(
	context.Context,
) ([]generated.Order, error) {
	panic("not implemented")
}

func (m *mockOrderRepository) UpdateOrderForModify(
	context.Context,
	generated.UpdateOrderForModifyParams,
) error {
	panic("not implemented")
}

func TestSnapshotRecoveryReplaysWALAfterSnapshotSequence(t *testing.T) {
	snapshotDir := t.TempDir()
	walDir := t.TempDir()

	snapshotSerializer := snapshot.NewJSONSerializer()
	walSerializer := wal.NewJSONSerializer()

	snapshotWriter := snapshot.NewWriter(
		snapshotDir,
		snapshotSerializer,
	)

	walManager := wal.NewManager(
		walDir,
		walSerializer,
	)

	reg := registry.New(
		snapshotWriter,
		walManager,
	)

	defer reg.Shutdown()

	// Get the engine for BTCUSDT.
	engine := reg.Get("BTCUSDT")

	// Create the order that exists in the snapshot.
	snapshotOrder := &order.Order{
		ID:        100,
		UserID:    1,
		Symbol:    "BTCUSDT",
		Side:      constants.OrderSideBuy,
		Type:      constants.OrderTypeLimit,
		Status:    constants.OrderStatusOpen,
		Price:     100000,
		Quantity:  10,
		Remaining: 10,
		Filled:    0,
	}

	// Snapshot represents engine state at sequence 10.
	snap := &snapshot.Snapshot{
		Symbol:         "BTCUSDT",
		Sequence:       10,
		LastTradePrice: 0,
		CreatedAt:      time.Now(),
		ActiveOrders: []*order.Order{
			snapshotOrder,
		},
	}

	err := snapshotWriter.Write(snap)
	assert.NoError(t, err)

	// WAL event at sequence 9.
	// This MUST NOT be replayed.
	oldOrder := &order.Order{
		ID:        90,
		UserID:    2,
		Symbol:    "BTCUSDT",
		Side:      constants.OrderSideBuy,
		Type:      constants.OrderTypeLimit,
		Status:    constants.OrderStatusOpen,
		Price:     99000,
		Quantity:  1,
		Remaining: 1,
	}

	writer, err := walManager.Writer("BTCUSDT")
	assert.NoError(t, err)

	err = writer.Write(
		wal.NewSubmitEvent(
			9,
			"BTCUSDT",
			oldOrder,
		),
	)
	assert.NoError(t, err)

	// WAL event at sequence 10.
	// This MUST NOT be replayed because the snapshot
	// already contains state through sequence 10.
	sameSequenceOrder := &order.Order{
		ID:        101,
		UserID:    3,
		Symbol:    "BTCUSDT",
		Side:      constants.OrderSideBuy,
		Type:      constants.OrderTypeLimit,
		Status:    constants.OrderStatusOpen,
		Price:     98000,
		Quantity:  2,
		Remaining: 2,
	}

	err = writer.Write(
		wal.NewSubmitEvent(
			10,
			"BTCUSDT",
			sameSequenceOrder,
		),
	)
	assert.NoError(t, err)

	// WAL event at sequence 11.
	// This MUST be replayed.
	newOrder := &order.Order{
		ID:        102,
		UserID:    4,
		Symbol:    "BTCUSDT",
		Side:      constants.OrderSideBuy,
		Type:      constants.OrderTypeLimit,
		Status:    constants.OrderStatusOpen,
		Price:     97000,
		Quantity:  3,
		Remaining: 3,
	}

	err = writer.Write(
		wal.NewSubmitEvent(
			11,
			"BTCUSDT",
			newOrder,
		),
	)
	assert.NoError(t, err)

	err = writer.Close()
	assert.NoError(t, err)

	loader := snapshot.NewLoader(
		snapshotDir,
		snapshotSerializer,
	)

	recovery := recovery.NewSnapshotRecovery(
		loader,
		reg,
		walManager,
	)

	restored, err := recovery.Restore("BTCUSDT")

	assert.NoError(t, err)
	assert.True(t, restored)

	// Snapshot order must exist.
	assert.NotNil(
		t,
		engine.OrderBook().GetOrder(100),
	)

	// Sequence 9 must NOT exist.
	assert.Nil(
		t,
		engine.OrderBook().GetOrder(90),
	)

	// Sequence 10 must NOT exist.
	assert.Nil(
		t,
		engine.OrderBook().GetOrder(101),
	)

	// Sequence 11 MUST exist.
	assert.NotNil(
		t,
		engine.OrderBook().GetOrder(102),
	)

	assert.Len(
		t,
		engine.OrderBook().Orders,
		2,
	)
}

func TestSnapshotRecoveryReplaysWALCancelAfterSnapshot(t *testing.T) {
	snapshotDir := t.TempDir()
	walDir := t.TempDir()

	snapshotSerializer := snapshot.NewJSONSerializer()
	walSerializer := wal.NewJSONSerializer()

	snapshotWriter := snapshot.NewWriter(
		snapshotDir,
		snapshotSerializer,
	)

	walManager := wal.NewManager(
		walDir,
		walSerializer,
	)

	reg := registry.New(
		snapshotWriter,
		walManager,
	)

	defer reg.Shutdown()

	orderToCancel := &order.Order{
		ID:        200,
		UserID:    1,
		Symbol:    "BTCUSDT",
		Side:      constants.OrderSideBuy,
		Type:      constants.OrderTypeLimit,
		Status:    constants.OrderStatusOpen,
		Price:     100000,
		Quantity:  10,
		Remaining: 10,
		Filled:    0,
	}

	snap := &snapshot.Snapshot{
		Symbol:         "BTCUSDT",
		Sequence:       10,
		LastTradePrice: 0,
		CreatedAt:      time.Now(),
		ActiveOrders: []*order.Order{
			orderToCancel,
		},
	}

	err := snapshotWriter.Write(snap)
	assert.NoError(t, err)

	writer, err := walManager.Writer("BTCUSDT")
	assert.NoError(t, err)

	err = writer.Write(
		wal.NewCancelEvent(
			11,
			"BTCUSDT",
			200,
		),
	)
	assert.NoError(t, err)

	err = writer.Close()
	assert.NoError(t, err)

	loader := snapshot.NewLoader(
		snapshotDir,
		snapshotSerializer,
	)

	recovery := recovery.NewSnapshotRecovery(
		loader,
		reg,
		walManager,
	)

	restored, err := recovery.Restore("BTCUSDT")

	assert.NoError(t, err)
	assert.True(t, restored)

	engine := reg.Get("BTCUSDT")

	assert.Nil(
		t,
		engine.OrderBook().GetOrder(200),
	)

	assert.Len(
		t,
		engine.OrderBook().Orders,
		0,
	)

	assert.Equal(
		t,
		uint64(11),
		engine.Sequence(),
	)
}

func TestSnapshotRecoveryReplaysWALModifyAfterSnapshot(t *testing.T) {
	snapshotDir := t.TempDir()
	walDir := t.TempDir()

	snapshotSerializer := snapshot.NewJSONSerializer()
	walSerializer := wal.NewJSONSerializer()

	snapshotWriter := snapshot.NewWriter(
		snapshotDir,
		snapshotSerializer,
	)

	walManager := wal.NewManager(
		walDir,
		walSerializer,
	)

	reg := registry.New(
		snapshotWriter,
		walManager,
	)

	defer reg.Shutdown()

	snapshotOrder := &order.Order{
		ID:        300,
		UserID:    1,
		Symbol:    "BTCUSDT",
		Side:      constants.OrderSideBuy,
		Type:      constants.OrderTypeLimit,
		Status:    constants.OrderStatusOpen,
		Price:     100000,
		Quantity:  10,
		Remaining: 10,
		Filled:    0,
	}

	snap := &snapshot.Snapshot{
		Symbol:         "BTCUSDT",
		Sequence:       10,
		LastTradePrice: 0,
		CreatedAt:      time.Now(),
		ActiveOrders: []*order.Order{
			snapshotOrder,
		},
	}

	err := snapshotWriter.Write(snap)
	assert.NoError(t, err)

	writer, err := walManager.Writer("BTCUSDT")
	assert.NoError(t, err)

	err = writer.Write(
		wal.NewModifyEvent(
			11,
			"BTCUSDT",
			300,
			105000,
			7,
		),
	)
	assert.NoError(t, err)

	err = writer.Close()
	assert.NoError(t, err)

	loader := snapshot.NewLoader(
		snapshotDir,
		snapshotSerializer,
	)

	recovery := recovery.NewSnapshotRecovery(
		loader,
		reg,
		walManager,
	)

	restored, err := recovery.Restore("BTCUSDT")

	assert.NoError(t, err)
	assert.True(t, restored)

	engine := reg.Get("BTCUSDT")

	recoveredOrder := engine.OrderBook().GetOrder(300)

	assert.NotNil(t, recoveredOrder)

	assert.Equal(
		t,
		int64(105000),
		recoveredOrder.Price,
	)

	assert.Equal(
		t,
		int64(7),
		recoveredOrder.Quantity,
	)

	assert.Equal(
		t,
		int64(7),
		recoveredOrder.Remaining,
	)

	assert.Len(
		t,
		engine.OrderBook().Orders,
		1,
	)
}
