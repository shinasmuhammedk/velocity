package recovery_test

import (
	"context"
	"testing"
	"time"

	"velocity/internal/engine/recovery"
	"velocity/internal/engine/registry"
	"velocity/internal/engine/snapshot"
	"velocity/internal/persistence/postgres/generated"

	"github.com/google/uuid"
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

	reg := registry.New(writer)

	repo := &mockOrderRepository{
		orders: []generated.Order{
			{
				ID:          uuid.New(),
				UserID:      uuid.New(),
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

	reg := registry.New(writer)

	repo := &mockOrderRepository{
		orders: []generated.Order{
			{
				ID:          uuid.New(),
				UserID:      uuid.New(),
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