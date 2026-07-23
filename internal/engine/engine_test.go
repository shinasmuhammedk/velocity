package engine_test

import (
	"testing"
	"time"

	"velocity/internal/domain/order"
	"velocity/internal/engine"
	"velocity/pkg/constants"
	"velocity/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlaceOrderFullFill(t *testing.T) {
	e := engine.New("BTCUSDT",nil,nil)

	sell := &order.Order{
		ID:          "sell-1",
		UserID:      "seller-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideSell,
		Type:        constants.OrderTypeLimit,
		TimeInForce: constants.TimeInForceGTC,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    100,
		Remaining:   100,
		CreatedAt:   time.Now(),
	}

	err := e.SubmitOrder(sell)
	require.NoError(t, err)

	buy := &order.Order{
		ID:          "buy-1",
		UserID:      "buyer-1",
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

	err = e.SubmitOrder(buy)
	require.NoError(t, err)

	var tradeReceived bool

	select {
	case trade := <-e.Trades():
		tradeReceived = true
		assert.Equal(t, int64(100), trade.Quantity)

	case <-time.After(time.Second):
		t.Fatal("trade timeout")
	}

	assert.True(t, tradeReceived)

	assert.Equal(
		t,
		constants.OrderStatusFilled,
		buy.Status,
	)

	assert.Equal(
		t,
		constants.OrderStatusFilled,
		sell.Status,
	)

	assert.Nil(t, e.OrderBook().BestAsk())
	assert.Nil(t, e.OrderBook().BestBid())
}

func TestPlaceOrderRestingOrder(t *testing.T) {
	e := engine.New("BTCUSDT",nil,nil)

	buy := &order.Order{
		ID:          "buy-1",
		UserID:      "buyer-1",
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

	err := e.SubmitOrder(buy)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return e.OrderBook().BestBid() != nil
	}, 100*time.Millisecond, 5*time.Millisecond)

	bestBid := e.OrderBook().BestBid()

	require.NotNil(t, bestBid)

	assert.Equal(t, int64(1000), bestBid.Price)
	assert.Equal(t, 1, bestBid.Size())
}

func TestPlaceOrderPartialFill(t *testing.T) {
	e := engine.New("BTCUSDT",nil,nil)

	sell := &order.Order{
		ID:          "sell-1",
		UserID:      "seller-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideSell,
		Type:        constants.OrderTypeLimit,
		TimeInForce: constants.TimeInForceGTC,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    50,
		Remaining:   50,
		CreatedAt:   time.Now(),
	}

	err := e.SubmitOrder(sell)
	require.NoError(t, err)

	buy := &order.Order{
		ID:          "buy-1",
		UserID:      "buyer-1",
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

	err = e.SubmitOrder(buy)
	require.NoError(t, err)

	select {
	case tr := <-e.Trades():
		assert.Equal(t, int64(50), tr.Quantity)

	case <-time.After(time.Second):
		t.Fatal("trade timeout")
	}

	assert.Equal(
		t,
		constants.OrderStatusPartiallyFilled,
		buy.Status,
	)

	assert.Equal(
		t,
		constants.OrderStatusFilled,
		sell.Status,
	)

	assert.Equal(t, int64(50), buy.Remaining)

	bestBid := e.OrderBook().BestBid()

	require.NotNil(t, bestBid)

	assert.Equal(t, int64(1000), bestBid.Price)
}


func TestEngineCancelOrder(t *testing.T) {
	e := engine.New("BTCUSDT",nil,nil)

	o := &order.Order{
		ID:          "1",
		UserID:      "user-1",
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

	e.OrderBook().AddOrder(o)

	err := e.CancelOrder(o.ID)

	require.NoError(t, err)

	assert.Equal(t, constants.OrderStatusCancelled, o.Status)
	assert.Nil(t, e.OrderBook().BestBid())
}

func TestEngineCancelUnknownOrder(t *testing.T) {
	e := engine.New("BTCUSDT",nil,nil)

	err := e.CancelOrder("does-not-exist")

	require.Error(t, err)
	assert.Equal(t, errors.ErrOrderNotFound, err)
}

func TestEngineCancelRemovesPriceLevel(t *testing.T) {
	e := engine.New("BTCUSDT",nil,nil)

	o := &order.Order{
		ID:          "1",
		UserID:      "user-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideSell,
		Type:        constants.OrderTypeLimit,
		TimeInForce: constants.TimeInForceGTC,
		Status:      constants.OrderStatusOpen,
		Price:       1010,
		Quantity:    100,
		Remaining:   100,
		CreatedAt:   time.Now(),
	}

	e.OrderBook().AddOrder(o)

	require.NotNil(t, e.OrderBook().BestAsk())

	err := e.CancelOrder(o.ID)

	require.NoError(t, err)

	assert.Nil(t, e.OrderBook().BestAsk())
}


func TestPostOnlyMarketOrderRejected(t *testing.T) {
	engine := engine.New("BTCUSDT",nil,nil)

	err := engine.SubmitOrder(&order.Order{
		ID:          "1",
		UserID:      "user1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeMarket,
		TimeInForce: constants.TimeInForcePostOnly,
		Quantity:    10,
		Remaining:   10,
	})

	require.Error(t, err)
}