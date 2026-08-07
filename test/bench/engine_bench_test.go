// test/bench/engine_bench_test.go
package bench

import (
	"testing"
	"time"

	"velocity/internal/domain/order"
	"velocity/internal/engine"
	"velocity/pkg/constants"
)

func createSellOrder(id int64, price int64, qty int64) *order.Order {
	return &order.Order{
		ID:          id,
		UserID:      201,
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideSell,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       price,
		Quantity:    qty,
		Remaining:   qty,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}
}

func createBuyOrder(id int64, price int64, qty int64) *order.Order {
	return &order.Order{
		ID:          id,
		UserID:      101,
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideBuy,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       price,
		Quantity:    qty,
		Remaining:   qty,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}
}

// buyIDOffset keeps benchmark buy-order IDs from ever colliding with
// seed sell-order IDs, regardless of how many seed levels are used or
// how large b.N grows.
const buyIDOffset = 1_000_000_000

func BenchmarkEngineMatching(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()

	e := engine.New("BTCUSDT", nil, nil)

	b.Cleanup(func() {
		e.Stop()
	})

	seed := createSellOrder(
		1,
		1000,
		1_000_000_000_000,
	)

	_ = e.SubmitOrder(seed)

	for {
		if e.OrderBook().BestAskPrice() == 1000 {
			break
		}
	}

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		buy := createBuyOrder(
			buyIDOffset+int64(i),
			1000,
			10,
		)

		_ = e.SubmitOrder(buy)

		<-e.Trades()
	}
}

func BenchmarkEngineMatchingDeepBook(b *testing.B) {
	const levels = 5000

	b.ReportAllocs()
	b.StopTimer()

	e := engine.New("BTCUSDT", nil, nil)

	b.Cleanup(func() {
		e.Stop()
	})

	for i := 0; i < levels; i++ {
		price := int64(1000 + i)

		sell := createSellOrder(
			int64(i+1),
			price,
			1_000_000_000,
		)

		_ = e.SubmitOrder(sell)
	}

	for {
		if e.OrderBook().BestAskPrice() == 1000 {
			break
		}
	}

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		buy := createBuyOrder(
			buyIDOffset+int64(i),
			5999,
			10,
		)

		_ = e.SubmitOrder(buy)

		<-e.Trades()
	}
}