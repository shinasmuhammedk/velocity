// test/bench/engine_bench_test.go
package bench

import (
	"fmt"
	"testing"
	"time"

	"velocity/internal/domain/order"
	"velocity/internal/engine"
	"velocity/pkg/constants"
)

func BenchmarkEngineMatching(b *testing.B) {
	e := engine.New("BTCUSDT")

	giantSell := &order.Order{
		ID:          "seed-giant-sell",
		UserID:      "seller-giant",
		Symbol:      "BTCUSDT",
		Side:        constants.OrderSideSell,
		Type:        constants.OrderTypeLimit,
		Status:      constants.OrderStatusOpen,
		Price:       1000,
		Quantity:    1_000_000_000_000,
		Remaining:   1_000_000_000_000,
		TimeInForce: constants.TimeInForceGTC,
		CreatedAt:   time.Now(),
	}
	e.SubmitOrder(giantSell)

	time.Sleep(50 * time.Millisecond)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buy := &order.Order{
			ID:          fmt.Sprintf("bench-buy-%d", i),
			UserID:      "bench-buyer",
			Symbol:      "BTCUSDT",
			Side:        constants.OrderSideBuy,
			Type:        constants.OrderTypeLimit,
			Status:      constants.OrderStatusOpen,
			Price:       1000,
			Quantity:    10,
			Remaining:   10,
			TimeInForce: constants.TimeInForceGTC,
			CreatedAt:   time.Now(),
		}
		e.SubmitOrder(buy)
		<-e.Trades()
	}
}


// BenchmarkEngineMatchingDeepBook simulates a realistic book: 5,000 distinct
// price levels, each with unlimited liquidity — stresses the heap's lazy
// cleanup and the linear FindFirst scan within a level, without ever
// running out of shares to match against.
func BenchmarkEngineMatchingDeepBook(b *testing.B) {
	e := engine.New("BTCUSDT")

	const numLevels = 5000

	// One resting order per price level, each with huge quantity —
	// gives real depth (5000 distinct levels to walk/skip through)
	// without any level ever emptying out mid-benchmark.
	for lvl := 0; lvl < numLevels; lvl++ {
		price := int64(1000 + lvl) // 5000 distinct ask prices: 1000..5999

		sell := &order.Order{
			ID:          fmt.Sprintf("seed-sell-%d", lvl),
			UserID:      fmt.Sprintf("seller-%d", lvl),
			Symbol:      "BTCUSDT",
			Side:        constants.OrderSideSell,
			Type:        constants.OrderTypeLimit,
			Status:      constants.OrderStatusOpen,
			Price:       price,
			Quantity:    1_000_000_000,
			Remaining:   1_000_000_000,
			TimeInForce: constants.TimeInForceGTC,
			CreatedAt:   time.Now(),
		}
		e.SubmitOrder(sell)
	}

	// 5,000 seed orders — give the engine time to process them all
	time.Sleep(1 * time.Second)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Willing to pay up to 5999 — the cheapest level (1000) will
		// always satisfy it immediately, but the heap still has to
		// manage 5000 live price levels underneath.
		buy := &order.Order{
			ID:          fmt.Sprintf("bench-buy-%d", i),
			UserID:      "bench-buyer",
			Symbol:      "BTCUSDT",
			Side:        constants.OrderSideBuy,
			Type:        constants.OrderTypeLimit,
			Status:      constants.OrderStatusOpen,
			Price:       5999,
			Quantity:    10,
			Remaining:   10,
			TimeInForce: constants.TimeInForceGTC,
			CreatedAt:   time.Now(),
		}
		e.SubmitOrder(buy)
		<-e.Trades()
	}
}