// test/bench/engine_bench_test.go
package bench

import (
	"testing"
	"time"

	"velocity/internal/domain/order"
	"velocity/internal/engine"
	"velocity/internal/engine/matcher"
	"velocity/internal/engine/orderbook"
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

func BenchmarkMatcherDirect(b *testing.B) {
	b.ReportAllocs()

	book := orderbook.New("BTCUSDT")
	m := matcher.New(book)

	sell := createSellOrder(
		1,
		1000,
		1_000_000_000_000,
	)

	book.AddOrder(sell)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buy := createBuyOrder(
			buyIDOffset+int64(i),
			1000,
			10,
		)

		trades, _ := m.Match(buy)

		if len(trades) != 1 {
			b.Fatal("expected one trade")
		}
	}
}

func BenchmarkEnginePipeline(b *testing.B) {
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

	done := make(chan struct{})

	go func() {
		defer close(done)

		for range e.Trades() {
			// Drain trades.
		}
	}()

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		buy := createBuyOrder(
			buyIDOffset+int64(i),
			1000,
			10,
		)

		_ = e.SubmitOrder(buy)
	}

	b.StopTimer()

	e.Stop()

	<-done
}

func BenchmarkEngineSubmitOnly(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()

	e := engine.New("BTCUSDT", nil, nil)

	b.Cleanup(func() {
		e.Stop()
	})

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		sell := createSellOrder(
			buyIDOffset+int64(i),
			1000+int64(i),
			1,
		)

		if err := e.SubmitOrder(sell); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMatcherMultiFill(b *testing.B) {
	const levels = 100

	b.ReportAllocs()
	b.StopTimer()

	// Prepare one complete book.
	book := orderbook.New("BTCUSDT")
	m := matcher.New(book)

	for i := 0; i < levels; i++ {
		sell := createSellOrder(
			int64(i+1),
			int64(1000+i),
			10,
		)

		sell.UserID = int64(200 + i)

		book.AddOrder(sell)
	}

	// Verify benchmark setup.
	if book.BestAskPrice() != 1000 {
		b.Fatalf(
			"expected best ask 1000, got %d",
			book.BestAskPrice(),
		)
	}

	b.StartTimer()

	// One measured multi-fill.
	buy := createBuyOrder(
		buyIDOffset,
		1099,
		levels*10,
	)

	trades, err := m.Match(buy)
	if err != nil {
		b.Fatal(err)
	}

	b.StopTimer()

	if len(trades) != levels {
		b.Fatalf(
			"expected %d trades, got %d",
			levels,
			len(trades),
		)
	}
}

func TestMultiFillDebug(t *testing.T) {
	book := orderbook.New("BTCUSDT")
	m := matcher.New(book)

	for i := 0; i < 100; i++ {
		sell := createSellOrder(
			int64(i+1),
			int64(1000+i),
			10,
		)

		sell.UserID = int64(200 + i)

		book.AddOrder(sell)
	}

	t.Logf("best ask: %d", book.BestAskPrice())

	buy := createBuyOrder(
		buyIDOffset,
		1099,
		1000,
	)

	trades, err := m.Match(buy)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("trades: %d", len(trades))
	t.Logf("remaining: %d", buy.Remaining)
}
