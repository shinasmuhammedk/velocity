package orderbook_test

import (
	"testing"
	"time"

	"velocity/internal/domain/order"
	"velocity/internal/engine/orderbook"
	"velocity/pkg/constants"
	"velocity/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newOrder(
	id string,
	side constants.OrderSide,
	price int64,
) *order.Order {
	return &order.Order{
		ID:          id,
		UserID:      "user-" + id,
		Symbol:      "BTCUSDT",
		Side:        side,
		Type:        constants.OrderTypeLimit,
		TimeInForce: constants.TimeInForceGTC,
		Status:      constants.OrderStatusOpen,
		Price:       price,
		Quantity:    100,
		Remaining:   100,
		CreatedAt:   time.Now(),
	}
}

func TestBestBid(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	book.AddOrder(newOrder("1", constants.OrderSideBuy, 1000))
	book.AddOrder(newOrder("2", constants.OrderSideBuy, 1010))
	book.AddOrder(newOrder("3", constants.OrderSideBuy, 1005))

	best := book.BestBid()

	require.NotNil(t, best)
	assert.Equal(t, int64(1010), best.Price)
}

func TestBestAsk(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	book.AddOrder(newOrder("1", constants.OrderSideSell, 1020))
	book.AddOrder(newOrder("2", constants.OrderSideSell, 1010))
	book.AddOrder(newOrder("3", constants.OrderSideSell, 1030))

	best := book.BestAsk()

	require.NotNil(t, best)
	assert.Equal(t, int64(1010), best.Price)
}

func TestRemoveBidLevel(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	book.AddOrder(newOrder("1", constants.OrderSideBuy, 1000))

	require.NotNil(t, book.BestBid())

	book.RemoveBidLevel(1000)

	assert.Nil(t, book.BestBid())
}

func TestRemoveAskLevel(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	book.AddOrder(newOrder("1", constants.OrderSideSell, 1000))

	require.NotNil(t, book.BestAsk())

	book.RemoveAskLevel(1000)

	assert.Nil(t, book.BestAsk())
}

func TestMultipleOrdersSamePriceLevel(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	book.AddOrder(newOrder("1", constants.OrderSideBuy, 1000))
	book.AddOrder(newOrder("2", constants.OrderSideBuy, 1000))
	book.AddOrder(newOrder("3", constants.OrderSideBuy, 1000))

	level := book.Bids[1000]

	require.NotNil(t, level)

	assert.Equal(t, 3, level.Size())
}


func TestCancelOrder(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	o := newOrder("1", constants.OrderSideBuy, 1000)

	book.AddOrder(o)

	err := book.CancelOrder(o.ID)

	require.NoError(t, err)

	assert.Equal(t, constants.OrderStatusCancelled, o.Status)
	assert.Nil(t, book.BestBid())
}


func TestCancelUnknownOrder(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	err := book.CancelOrder("does-not-exist")

	require.Error(t, err)
	assert.Equal(t, errors.ErrOrderNotFound, err)
}

func TestCancelAlreadyCancelledOrder(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	o := newOrder("1", constants.OrderSideBuy, 1000)

	book.AddOrder(o)

	err := book.CancelOrder(o.ID)
	require.NoError(t, err)

	err = book.CancelOrder(o.ID)

	require.Error(t, err)
	assert.Equal(t, errors.ErrOrderNotFound, err)
}

func TestCancelRemovesPriceLevel(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	o := newOrder("1", constants.OrderSideSell, 1010)

	book.AddOrder(o)

	require.NotNil(t, book.BestAsk())

	err := book.CancelOrder(o.ID)

	require.NoError(t, err)

	assert.Nil(t, book.BestAsk())
}


func TestOrderIndexedOnAdd(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	o := newOrder("1", constants.OrderSideBuy, 1000)

	book.AddOrder(o)

	_, exists := book.Orders[o.ID]

	assert.True(t, exists)
}

func TestOrderRemovedFromIndexOnCancel(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	o := newOrder("1", constants.OrderSideBuy, 1000)

	book.AddOrder(o)

	err := book.CancelOrder(o.ID)

	require.NoError(t, err)

	_, exists := book.Orders[o.ID]

	assert.False(t, exists)
}

func TestModifyOrderQuantityReductionKeepsPriority(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	o := newOrder("1", constants.OrderSideBuy, 1000)

	book.AddOrder(o)

	err := book.ModifyOrder(
		o.ID,
		1000,
		50,
	)

	require.NoError(t, err)

	assert.Equal(t, int64(50), o.Quantity)
	assert.Equal(t, int64(50), o.Remaining)

	level := book.Bids[1000]

	require.NotNil(t, level)

	first := level.Orders.Front().Value.(*order.Order)

	assert.Equal(t, o.ID, first.ID)
}

func TestModifyOrderPriceChangeLosesPriority(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	o1 := newOrder("1", constants.OrderSideBuy, 1000)
	o2 := newOrder("2", constants.OrderSideBuy, 1010)

	book.AddOrder(o1)
	book.AddOrder(o2)

	err := book.ModifyOrder(
		o1.ID,
		1010,
		100,
	)

	require.NoError(t, err)

	level := book.Bids[1010]

	require.NotNil(t, level)

	first := level.Orders.Front().Value.(*order.Order)

	assert.Equal(t, o2.ID, first.ID)
}

func TestModifyOrderQuantityIncreaseLosesPriority(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	o1 := newOrder("1", constants.OrderSideBuy, 1000)
	o2 := newOrder("2", constants.OrderSideBuy, 1000)

	book.AddOrder(o1)
	book.AddOrder(o2)

	err := book.ModifyOrder(
		o1.ID,
		1000,
		200,
	)

	require.NoError(t, err)

	level := book.Bids[1000]

	require.NotNil(t, level)

	first := level.Orders.Front().Value.(*order.Order)

	assert.Equal(t, o2.ID, first.ID)
}

func TestModifyFilledOrderFails(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	o := newOrder("1", constants.OrderSideBuy, 1000)
	o.Status = constants.OrderStatusFilled

	book.AddOrder(o)

	err := book.ModifyOrder(
		o.ID,
		1000,
		50,
	)

	require.Error(t, err)
	assert.Equal(t, errors.ErrOrderFilled, err)
}

func TestModifyCancelledOrderFails(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	o := newOrder("1", constants.OrderSideBuy, 1000)
	o.Status = constants.OrderStatusCancelled

	book.AddOrder(o)

	err := book.ModifyOrder(
		o.ID,
		1000,
		50,
	)

	require.Error(t, err)
	assert.Equal(t, errors.ErrOrderCancelled, err)
}

func TestModifyUnknownOrderFails(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	err := book.ModifyOrder(
		"unknown",
		1000,
		100,
	)

	require.Error(t, err)
	assert.Equal(t, errors.ErrOrderNotFound, err)
}

func TestModifyQuantityBelowFilledFails(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	o := newOrder("1", constants.OrderSideBuy, 1000)

	o.Filled = 80
	o.Remaining = 20

	book.AddOrder(o)

	err := book.ModifyOrder(
		o.ID,
		1000,
		50,
	)

	require.Error(t, err)
	assert.Equal(t, errors.ErrInvalidOrder, err)
}

func TestModifyCreatesNewPriceLevel(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	o := newOrder("1", constants.OrderSideBuy, 1000)

	book.AddOrder(o)

	err := book.ModifyOrder(
		o.ID,
		1010,
		100,
	)

	require.NoError(t, err)

	assert.Nil(t, book.Bids[1000])
	assert.NotNil(t, book.Bids[1010])
}

func TestModifyRemovesOldPriceLevel(t *testing.T) {
	book := orderbook.New("BTCUSDT")

	o := newOrder("1", constants.OrderSideBuy, 1000)

	book.AddOrder(o)

	err := book.ModifyOrder(
		o.ID,
		1010,
		100,
	)

	require.NoError(t, err)

	_, exists := book.Bids[1000]

	assert.False(t, exists)
}


