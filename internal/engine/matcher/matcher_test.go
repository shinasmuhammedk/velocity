package matcher_test

import (
	"testing"
	"time"

	"velocity/internal/domain/order"
	"velocity/internal/engine/matcher"
	"velocity/internal/engine/orderbook"
	"velocity/pkg/constants"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchSellOrderFullFill(t *testing.T) {
	book := orderbook.New("BTCUSDT")
	m := matcher.New(book)

	buy := &order.Order{
		ID:          "buy-1",
		UserID:      "buyer-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    100,
		Remaining:   100,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	book.AddOrder(buy)

	sell := &order.Order{
		ID:          "sell-1",
		UserID:      "seller-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideSell,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    100,
		Remaining:   100,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	trades, err := m.Match(sell)

	require.NoError(t, err)
	require.Len(t, trades, 1)

	assert.Equal(t, int64(100), trades[0].Quantity)
	assert.Equal(t, int64(1000), trades[0].Price)

	assert.Equal(t, int64(0), sell.Remaining)
	assert.Equal(t, int64(0), buy.Remaining)

	assert.Equal(t, constants.OrderStatusFilled, sell.Status)
	assert.Equal(t, constants.OrderStatusFilled, buy.Status)

	assert.Nil(t, book.BestBid())
}

func TestMatchSellOrderPartialFill(t *testing.T) {
	book := orderbook.New("BTCUSDT")
	m := matcher.New(book)

	buy := &order.Order{
		ID:          "buy-1",
		UserID:      "buyer-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    50,
		Remaining:   50,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	book.AddOrder(buy)

	sell := &order.Order{
		ID:          "sell-1",
		UserID:      "seller-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideSell,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    100,
		Remaining:   100,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	trades, err := m.Match(sell)

	require.NoError(t, err)
	require.Len(t, trades, 1)

	assert.Equal(t, int64(50), trades[0].Quantity)

	assert.Equal(t, int64(50), sell.Remaining)
	assert.Equal(t, int64(0), buy.Remaining)

	assert.Equal(t, constants.OrderStatusPartiallyFilled, sell.Status)
	assert.Equal(t, constants.OrderStatusFilled, buy.Status)

	assert.NotNil(t, book.BestAsk())
}

func TestMatchSellOrderFIFO(t *testing.T) {
	book := orderbook.New("BTCUSDT")
	m := matcher.New(book)

	buy1 := &order.Order{
		ID:          "buy-1",
		UserID:      "buyer-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    50,
		Remaining:   50,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	buy2 := &order.Order{
		ID:          "buy-2",
		UserID:      "buyer-2",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    50,
		Remaining:   50,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now().Add(time.Second),
	}

	book.AddOrder(buy1)
	book.AddOrder(buy2)

	sell := &order.Order{
		ID:          "sell-1",
		UserID:      "seller-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideSell,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    50,
		Remaining:   50,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	trades, err := m.Match(sell)

	require.NoError(t, err)
	require.Len(t, trades, 1)

	// buy1 was added first — it should fill first, not buy2
	assert.Equal(t, "buy-1", trades[0].BuyOrderID)

	assert.Equal(t, int64(0), buy1.Remaining)
	assert.Equal(t, int64(50), buy2.Remaining)
}

func TestMatchSellOrderMultiplePriceLevels(t *testing.T) {
	book := orderbook.New("BTCUSDT")
	m := matcher.New(book)

	book.AddOrder(&order.Order{
		ID:          "buy-1",
		UserID:      "buyer-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1005, // best (highest) bid — sellers want the highest price
		Quantity:    50,
		Remaining:   50,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	})

	book.AddOrder(&order.Order{
		ID:          "buy-2",
		UserID:      "buyer-2",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    100,
		Remaining:   100,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	})

	sell := &order.Order{
		ID:          "sell-1",
		UserID:      "seller-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideSell,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000, // willing to accept as low as 1000, so both levels qualify
		Quantity:    120,
		Remaining:   120,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	trades, err := m.Match(sell)

	require.NoError(t, err)
	require.Len(t, trades, 2)

	// Best bid (1005) should be matched FIRST
	assert.Equal(t, int64(50), trades[0].Quantity)
	assert.Equal(t, int64(1005), trades[0].Price)

	assert.Equal(t, int64(70), trades[1].Quantity)
	assert.Equal(t, int64(1000), trades[1].Price)

	assert.Equal(t, int64(0), sell.Remaining)
}

func TestMatchSellOrderSkipsSelfTradeToNextLevel(t *testing.T) {
	book := orderbook.New("BTCUSDT")
	m := matcher.New(book)

	// seller-1's own resting buy order sits at the BEST price (1005).
	// It must be skipped due to self-trade prevention.
	ownBuy := &order.Order{
		ID:          "buy-own",
		UserID:      "seller-1", // same user as the incoming sell below
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1005,
		Quantity:    50,
		Remaining:   50,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	// A different buyer sits at a WORSE price (1000) — this is the one
	// that should actually get matched.
	otherBuy := &order.Order{
		ID:          "buy-other",
		UserID:      "buyer-2",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    50,
		Remaining:   50,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	book.AddOrder(ownBuy)
	book.AddOrder(otherBuy)

	sell := &order.Order{
		ID:          "sell-1",
		UserID:      "seller-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideSell,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000, // willing to accept as low as 1000, so both levels qualify
		Quantity:    50,
		Remaining:   50,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	trades, err := m.Match(sell)

	require.NoError(t, err)
	require.Len(t, trades, 1)

	// Must have matched the OTHER buyer at 1000, not self at 1005
	assert.Equal(t, "buy-other", trades[0].BuyOrderID)
	assert.Equal(t, int64(1000), trades[0].Price)
	assert.Equal(t, int64(50), trades[0].Quantity)

	assert.Equal(t, int64(0), sell.Remaining)
	assert.Equal(t, constants.OrderStatusFilled, sell.Status)

	// The self-order at 1005 must still be resting, untouched
	assert.Equal(t, int64(50), ownBuy.Remaining)
	assert.Equal(t, constants.OrderStatusOpen, ownBuy.Status)
}

func TestMatchMarketBuyOrderConsumesLiquidity(t *testing.T) {
	book := orderbook.New("BTCUSDT")
	m := matcher.New(book)

	sell := &order.Order{
		ID:          "sell-1",
		UserID:      "seller-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideSell,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    100,
		Remaining:   100,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	book.AddOrder(sell)

	// Market buy — Price is irrelevant/unused here (commonly 0),
	// should match regardless of the resting order's price.
	buy := &order.Order{
		ID:          "buy-1",
		UserID:      "buyer-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeMarket,
		Status:      constants.OrderStatusOpen,
		Price:       0,
		Quantity:    100,
		Remaining:   100,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	trades, err := m.Match(buy)

	require.NoError(t, err)
	require.Len(t, trades, 1)

	// Trade executes at the RESTING order's price, not the market order's price
	assert.Equal(t, int64(1000), trades[0].Price)
	assert.Equal(t, int64(100), trades[0].Quantity)

	assert.Equal(t, int64(0), buy.Remaining)
	assert.Equal(t, constants.OrderStatusFilled, buy.Status)
	assert.Nil(t, book.BestAsk())
}

func TestMatchMarketSellOrderConsumesLiquidity(t *testing.T) {
	book := orderbook.New("BTCUSDT")
	m := matcher.New(book)

	buy := &order.Order{
		ID:          "buy-1",
		UserID:      "buyer-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    100,
		Remaining:   100,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	book.AddOrder(buy)

	sell := &order.Order{
		ID:          "sell-1",
		UserID:      "seller-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideSell,
		Type:        constants.OrderTypeMarket,
		Status:      constants.OrderStatusOpen,
		Price:       0,
		Quantity:    100,
		Remaining:   100,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	trades, err := m.Match(sell)

	require.NoError(t, err)
	require.Len(t, trades, 1)

	assert.Equal(t, int64(1000), trades[0].Price)
	assert.Equal(t, int64(100), trades[0].Quantity)

	assert.Equal(t, int64(0), sell.Remaining)
	assert.Equal(t, constants.OrderStatusFilled, sell.Status)
	assert.Nil(t, book.BestBid())
}

func TestMatchMarketBuyOrderPartialFillDoesNotRest(t *testing.T) {
	book := orderbook.New("BTCUSDT")
	m := matcher.New(book)

	// Only 30 available, but the market order wants 100
	sell := &order.Order{
		ID:          "sell-1",
		UserID:      "seller-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideSell,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    30,
		Remaining:   30,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	book.AddOrder(sell)

	buy := &order.Order{
		ID:          "buy-1",
		UserID:      "buyer-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeMarket,
		Status:      constants.OrderStatusOpen,
		Price:       0,
		Quantity:    100,
		Remaining:   100,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	trades, err := m.Match(buy)

	require.NoError(t, err)
	require.Len(t, trades, 1)
	assert.Equal(t, int64(30), trades[0].Quantity)

	// 70 units unfilled — should be dropped, NOT resting on the book
	assert.Equal(t, int64(70), buy.Remaining)
	assert.Equal(t, constants.OrderStatusPartiallyFilled, buy.Status)

	// The book must have nothing resting — no ask (fully consumed) and,
	// critically, no bid either (the leftover market buy must NOT be added)
	assert.Nil(t, book.BestAsk())
	assert.Nil(t, book.BestBid())
}

func TestMatchMarketOrderOnEmptyBookProducesNoTrades(t *testing.T) {
	book := orderbook.New("BTCUSDT")
	m := matcher.New(book)

	buy := &order.Order{
		ID:          "buy-1",
		UserID:      "buyer-1",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeMarket,
		Status:      constants.OrderStatusOpen,
		Price:       0,
		Quantity:    100,
		Remaining:   100,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}

	trades, err := m.Match(buy)

	require.NoError(t, err)
	assert.Len(t, trades, 0)

	// Fully unfilled market order — must NOT rest on the book
	assert.Equal(t, int64(100), buy.Remaining)
	assert.Equal(t, constants.OrderStatusOpen, buy.Status)
	assert.Nil(t, book.BestBid())
}