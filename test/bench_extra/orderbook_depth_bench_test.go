package orderbook_test

import (
	"fmt"
	"testing"
	"time"

	"velocity/internal/domain/order"
	"velocity/internal/engine/orderbook"
	"velocity/pkg/constants"
)

func depthBenchmarkOrder(
	id int64,
	side constants.OrderSide,
	price int64,
) *order.Order {
	return &order.Order{
		ID:          id,
		UserID:      id + 100000,
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

// BenchmarkBestBidDepth measures BestBid performance as the number
// of price levels in the order book increases.
func BenchmarkBestBidDepth(b *testing.B) {
	depths := []int{10, 100, 1000, 10000}

	for _, depth := range depths {
		b.Run(fmt.Sprintf("Depth-%d", depth), func(b *testing.B) {
			book := orderbook.New("BTCUSDT")

			for i := 0; i < depth; i++ {
				book.AddOrder(
					depthBenchmarkOrder(
						int64(i+1),
						constants.OrderSideBuy,
						100000+int64(i),
					),
				)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = book.BestBid()
			}
		})
	}
}

// BenchmarkBestAskDepth measures BestAsk performance as the number
// of price levels in the order book increases.
func BenchmarkBestAskDepth(b *testing.B) {
	depths := []int{10, 100, 1000, 10000}

	for _, depth := range depths {
		b.Run(fmt.Sprintf("Depth-%d", depth), func(b *testing.B) {
			book := orderbook.New("BTCUSDT")

			for i := 0; i < depth; i++ {
				book.AddOrder(
					depthBenchmarkOrder(
						int64(i+1),
						constants.OrderSideSell,
						100000+int64(i),
					),
				)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = book.BestAsk()
			}
		})
	}
}

// BenchmarkBidLevelsDepth measures the cost of generating the top
// 20 bid levels at different book depths.
func BenchmarkBidLevelsDepth(b *testing.B) {
	depths := []int{10, 100, 1000, 10000}

	for _, depth := range depths {
		b.Run(fmt.Sprintf("Depth-%d", depth), func(b *testing.B) {
			book := orderbook.New("BTCUSDT")

			for i := 0; i < depth; i++ {
				book.AddOrder(
					depthBenchmarkOrder(
						int64(i+1),
						constants.OrderSideBuy,
						100000+int64(i),
					),
				)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = book.BidLevels(20)
			}
		})
	}
}

// BenchmarkAskLevelsDepth measures the cost of generating the top
// 20 ask levels at different book depths.
func BenchmarkAskLevelsDepth(b *testing.B) {
	depths := []int{10, 100, 1000, 10000}

	for _, depth := range depths {
		b.Run(fmt.Sprintf("Depth-%d", depth), func(b *testing.B) {
			book := orderbook.New("BTCUSDT")

			for i := 0; i < depth; i++ {
				book.AddOrder(
					depthBenchmarkOrder(
						int64(i+1),
						constants.OrderSideSell,
						100000+int64(i),
					),
				)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = book.AskLevels(20)
			}
		})
	}
}

// BenchmarkActiveOrdersDepth measures ActiveOrders performance as
// the total number of active orders increases.
func BenchmarkActiveOrdersDepth(b *testing.B) {
	depths := []int{10, 100, 1000, 10000}

	for _, depth := range depths {
		b.Run(fmt.Sprintf("Depth-%d", depth), func(b *testing.B) {
			book := orderbook.New("BTCUSDT")

			for i := 0; i < depth; i++ {
				book.AddOrder(
					depthBenchmarkOrder(
						int64(i+1),
						constants.OrderSideBuy,
						100000+int64(i),
					),
				)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = book.ActiveOrders()
			}
		})
	}
}