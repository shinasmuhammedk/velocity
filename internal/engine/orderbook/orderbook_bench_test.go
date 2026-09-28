package orderbook_test

import (
	"testing"
	"time"

	"velocity/internal/domain/order"
	"velocity/internal/engine/orderbook"
	"velocity/pkg/constants"
)

func benchmarkOrder(id int64, side constants.OrderSide, price int64) *order.Order {
	return &order.Order{
		ID:          id,
		UserID:      id + 1000,
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

func BenchmarkOrderBookAddOrder(b *testing.B) {
	book := orderbook.New("BTCUSDT")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		price := int64(100000 + i)
		book.AddOrder(
			benchmarkOrder(
				int64(i+1),
				constants.OrderSideBuy,
				price,
			),
		)
	}
}

func BenchmarkOrderBookBestBid(b *testing.B) {
	book := orderbook.New("BTCUSDT")

	for i := int64(0); i < 1000; i++ {
		book.AddOrder(
			benchmarkOrder(
				i+1,
				constants.OrderSideBuy,
				100000+i,
			),
		)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = book.BestBid()
	}
}

func BenchmarkOrderBookBestAsk(b *testing.B) {
	book := orderbook.New("BTCUSDT")

	for i := int64(0); i < 1000; i++ {
		book.AddOrder(
			benchmarkOrder(
				i+1,
				constants.OrderSideSell,
				100000+i,
			),
		)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = book.BestAsk()
	}
}

func BenchmarkOrderBookBestBidExcluding(b *testing.B) {
	book := orderbook.New("BTCUSDT")

	for i := int64(0); i < 1000; i++ {
		book.AddOrder(
			benchmarkOrder(
				i+1,
				constants.OrderSideBuy,
				100000+i,
			),
		)
	}

	excluded := make(map[int64]bool)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		excluded[100999] = true
		_ = book.BestBidExcluding(excluded)
		delete(excluded, 100999)
	}
}

func BenchmarkOrderBookBestAskExcluding(b *testing.B) {
	book := orderbook.New("BTCUSDT")

	for i := int64(0); i < 1000; i++ {
		book.AddOrder(
			benchmarkOrder(
				i+1,
				constants.OrderSideSell,
				100000+i,
			),
		)
	}

	excluded := make(map[int64]bool)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		excluded[100000] = true
		_ = book.BestAskExcluding(excluded)
		delete(excluded, 100000)
	}
}

func BenchmarkOrderBookCancelOrder(b *testing.B) {
	book := orderbook.New("BTCUSDT")

	for i := int64(0); i < int64(b.N); i++ {
		book.AddOrder(
			benchmarkOrder(
				i+1,
				constants.OrderSideBuy,
				100000,
			),
		)
	}

	b.ResetTimer()

	for i := int64(0); i < int64(b.N); i++ {
		_, _ = book.CancelOrder(i + 1)
	}
}

func BenchmarkOrderBookModifyQuantity(b *testing.B) {
	book := orderbook.New("BTCUSDT")

	o := benchmarkOrder(
		1,
		constants.OrderSideBuy,
		100000,
	)

	book.AddOrder(o)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = book.ModifyOrder(
			1,
			100000,
			100,
		)
	}
}

func BenchmarkOrderBookBidLevels(b *testing.B) {
	book := orderbook.New("BTCUSDT")

	for i := int64(0); i < 1000; i++ {
		book.AddOrder(
			benchmarkOrder(
				i+1,
				constants.OrderSideBuy,
				100000+i,
			),
		)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = book.BidLevels(20)
	}
}

func BenchmarkOrderBookAskLevels(b *testing.B) {
	book := orderbook.New("BTCUSDT")

	for i := int64(0); i < 1000; i++ {
		book.AddOrder(
			benchmarkOrder(
				i+1,
				constants.OrderSideSell,
				100000+i,
			),
		)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = book.AskLevels(20)
	}
}

func BenchmarkOrderBookActiveOrders(b *testing.B) {
	book := orderbook.New("BTCUSDT")

	for i := int64(0); i < 1000; i++ {
		book.AddOrder(
			benchmarkOrder(
				i+1,
				constants.OrderSideBuy,
				100000+i,
			),
		)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = book.ActiveOrders()
	}
}