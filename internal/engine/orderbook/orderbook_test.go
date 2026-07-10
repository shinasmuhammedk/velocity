package orderbook_test

import (
	"testing"
	"time"

	"velocity/internal/domain/order"
	"velocity/internal/engine/orderbook"
	"velocity/pkg/constants"

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