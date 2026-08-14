package recovery_test

import (
	"testing"
	"time"

	"velocity/internal/domain/order"
	"velocity/internal/engine/recovery"
	"velocity/internal/engine/registry"
	"velocity/internal/engine/snapshot"
	"velocity/internal/engine/wal"
	"velocity/pkg/constants"

	"github.com/stretchr/testify/assert"
)

func TestSnapshotRecoveryReplaysWALSubmit(t *testing.T) {
	snapshotDir := t.TempDir()
	walDir := t.TempDir()

	snapshotSerializer := snapshot.NewJSONSerializer()
	walSerializer := wal.NewJSONSerializer()

	snapshotWriter := snapshot.NewWriter(
		snapshotDir,
		snapshotSerializer,
	)

	snapshotLoader := snapshot.NewLoader(
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

	engine := reg.Get("BTCUSDT")

	// Order already present in the snapshot.
	snapshotOrder := &order.Order{
		ID:          1,
		UserID:      101,
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		TimeInForce: constants.TimeInForceGTC,
		Status:      constants.OrderStatusOpen,
		Price:       100000,
		Quantity:    10,
		Remaining:   10,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	engine.OrderBook().AddOrder(snapshotOrder)

	snap := &snapshot.Snapshot{
		Symbol:       "BTCUSDT",
		Sequence:     10,
		ActiveOrders: []*order.Order{snapshotOrder},
		StopOrders:   []*order.Order{},
	}

	assert.NoError(t, snapshotWriter.Write(snap))

	// WAL event after the snapshot.
	walWriter, err := walManager.Writer("BTCUSDT")
	assert.NoError(t, err)

	walOrder := &order.Order{
		ID:          2,
		UserID:      102,
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideSell,
		Type:        constants.OrderTypeLimit,
		TimeInForce: constants.TimeInForceGTC,
		Status:      constants.OrderStatusOpen,
		Price:       101000,
		Quantity:    5,
		Remaining:   5,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	event := wal.NewSubmitEvent(
		11,
		"BTCUSDT",
		walOrder,
	)

	assert.NoError(t, walWriter.Write(event))

	recovery := recovery.NewSnapshotRecovery(
		snapshotLoader,
		reg,
		walManager,
	)

	restored, err := recovery.Restore("BTCUSDT")

	assert.NoError(t, err)
	assert.True(t, restored)

	engine = reg.Get("BTCUSDT")

	assert.NotNil(t, engine.OrderBook().GetOrder(1))
	assert.NotNil(t, engine.OrderBook().GetOrder(2))
}

func TestSnapshotRecoveryReplaysWALCancel(t *testing.T) {
	snapshotDir := t.TempDir()
	walDir := t.TempDir()

	snapshotSerializer := snapshot.NewJSONSerializer()
	walSerializer := wal.NewJSONSerializer()

	snapshotWriter := snapshot.NewWriter(
		snapshotDir,
		snapshotSerializer,
	)

	snapshotLoader := snapshot.NewLoader(
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

	engine := reg.Get("BTCUSDT")

	snapshotOrder := &order.Order{
		ID:          10,
		UserID:      101,
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		TimeInForce: constants.TimeInForceGTC,
		Status:      constants.OrderStatusOpen,
		Price:       100000,
		Quantity:    10,
		Remaining:   10,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	engine.OrderBook().AddOrder(snapshotOrder)

	snap := &snapshot.Snapshot{
		Symbol:       "BTCUSDT",
		Sequence:     10,
		ActiveOrders: []*order.Order{snapshotOrder},
		StopOrders:   []*order.Order{},
	}

	assert.NoError(t, snapshotWriter.Write(snap))

	walWriter, err := walManager.Writer("BTCUSDT")
	assert.NoError(t, err)

	cancelEvent := wal.NewCancelEvent(
		11,
		"BTCUSDT",
		10,
	)

	assert.NoError(t, walWriter.Write(cancelEvent))

	r := recovery.NewSnapshotRecovery(
		snapshotLoader,
		reg,
		walManager,
	)

	restored, err := r.Restore("BTCUSDT")

	assert.NoError(t, err)
	assert.True(t, restored)

	engine = reg.Get("BTCUSDT")

	assert.Nil(
		t,
		engine.OrderBook().GetOrder(10),
	)
}

func TestSnapshotRecoveryReplaysWALModify(t *testing.T) {
	snapshotDir := t.TempDir()
	walDir := t.TempDir()

	snapshotSerializer := snapshot.NewJSONSerializer()
	walSerializer := wal.NewJSONSerializer()

	snapshotWriter := snapshot.NewWriter(
		snapshotDir,
		snapshotSerializer,
	)

	snapshotLoader := snapshot.NewLoader(
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

	engine := reg.Get("BTCUSDT")

	snapshotOrder := &order.Order{
		ID:          20,
		UserID:      201,
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		TimeInForce: constants.TimeInForceGTC,
		Status:      constants.OrderStatusOpen,
		Price:       100000,
		Quantity:    10,
		Remaining:   10,
		Filled:      0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	engine.OrderBook().AddOrder(snapshotOrder)

	snap := &snapshot.Snapshot{
		Symbol:       "BTCUSDT",
		Sequence:     10,
		ActiveOrders: []*order.Order{snapshotOrder},
		StopOrders:   []*order.Order{},
	}

	assert.NoError(t, snapshotWriter.Write(snap))

	walWriter, err := walManager.Writer("BTCUSDT")
	assert.NoError(t, err)

	modifyEvent := wal.NewModifyEvent(
		11,
		"BTCUSDT",
		20,
		105000,
		15,
	)

	assert.NoError(t, walWriter.Write(modifyEvent))

	r := recovery.NewSnapshotRecovery(
		snapshotLoader,
		reg,
		walManager,
	)

	restored, err := r.Restore("BTCUSDT")

	assert.NoError(t, err)
	assert.True(t, restored)

	engine = reg.Get("BTCUSDT")

	recoveredOrder := engine.OrderBook().GetOrder(20)

	assert.NotNil(t, recoveredOrder)

	assert.Equal(t, int64(105000), recoveredOrder.Price)
	assert.Equal(t, int64(15), recoveredOrder.Quantity)
	assert.Equal(t, int64(15), recoveredOrder.Remaining)
}
