package pricelevel_test

import (
	"testing"
	"time"

	"velocity/internal/domain/order"
	"velocity/internal/engine/pricelevel"
	"velocity/pkg/constants"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newOrder(id int64) *order.Order {
	return &order.Order{
		ID:          id,
		UserID:      id + 1000,
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		TimeInForce: constants.TimeInForceGTC,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    100,
		Remaining:   100,
		CreatedAt:   time.Now(),
	}
}

func TestPriceLevelAddAndFront(t *testing.T) {
	level := pricelevel.New(1000)

	order1 := newOrder(1)

	level.AddOrder(order1)

	front := level.Front()

	require.NotNil(t, front)
	assert.Equal(t, order1.ID, front.ID)
}

func TestPriceLevelFIFO(t *testing.T) {
	level := pricelevel.New(1000)

	order1 := newOrder(1)
	order2 := newOrder(2)
	order3 := newOrder(3)

	level.AddOrder(order1)
	level.AddOrder(order2)
	level.AddOrder(order3)

	assert.Equal(t, int64(1), level.Front().ID)

	level.RemoveFront()
	assert.Equal(t, int64(2), level.Front().ID)

	level.RemoveFront()
	assert.Equal(t, int64(3), level.Front().ID)
}

func TestPriceLevelRemoveFront(t *testing.T) {
	level := pricelevel.New(1000)

	order1 := newOrder(1)

	level.AddOrder(order1)

	assert.False(t, level.IsEmpty())

	level.RemoveFront()

	assert.True(t, level.IsEmpty())
	assert.Nil(t, level.Front())
}

func TestPriceLevelIsEmpty(t *testing.T) {
	level := pricelevel.New(1000)

	assert.True(t, level.IsEmpty())

	level.AddOrder(newOrder(1))

	assert.False(t, level.IsEmpty())
}