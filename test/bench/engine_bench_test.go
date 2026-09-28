// test/bench/engine_bench_test.go
package bench

import (
	"sync"
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

	stopDrain := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			select {
			case <-e.Trades():
				// Drain trades.
			case <-stopDrain:
				return
			}
		}
	}()

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		buy := createBuyOrder(
			buyIDOffset+int64(i),
			1000,
			10,
		)

		if err := e.SubmitOrder(buy); err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	close(stopDrain)
	<-done

	// e.Stop()
}
func BenchmarkEngineSubmitOnly(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()

	e := engine.New("BTCUSDT", nil, nil)

	b.Cleanup(func() {
		e.Stop()
	})

	// Use one fixed price so the benchmark does not create
	// a new PriceLevel on every iteration.
	const price int64 = 1000

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		sell := createSellOrder(
			buyIDOffset+int64(i),
			price,
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

	for i := 0; i < b.N; i++ {
		// Prepare the book outside the timed section.
		b.StopTimer()

		book := orderbook.New("BTCUSDT")
		m := matcher.New(book)

		for j := 0; j < levels; j++ {
			sell := createSellOrder(
				int64(j+1),
				int64(1000+j),
				10,
			)
			sell.UserID = int64(200 + j)
			book.AddOrder(sell)
		}

		buy := createBuyOrder(
			buyIDOffset+int64(i),
			1099,
			levels*10,
		)

		b.StartTimer()

		// Only the actual matching operation is measured.
		trades, err := m.Match(buy)

		b.StopTimer()

		if err != nil {
			b.Fatal(err)
		}

		if len(trades) != levels {
			b.Fatalf(
				"expected %d trades, got %d",
				levels,
				len(trades),
			)
		}
	}
}

func BenchmarkMatcherMultiFillAllocs(b *testing.B) {
	const levels = 100

	book := orderbook.New("BTCUSDT")
	m := matcher.New(book)

	for j := 0; j < levels; j++ {
		sell := createSellOrder(
			int64(j+1),
			int64(1000+j),
			10,
		)
		sell.UserID = int64(200 + j)
		book.AddOrder(sell)
	}

	buy := createBuyOrder(
		buyIDOffset,
		1099,
		levels*10,
	)

	allocs := testing.AllocsPerRun(1000, func() {
		// This benchmark only works if the matcher/book state
		// is restored between runs.
		_, _ = m.Match(buy)
	})

	b.ReportMetric(allocs, "allocs/op")
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

func BenchmarkEngineSubmitExistingOrder(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()

	e := engine.New("BTCUSDT", nil, nil)

	sell := createSellOrder(1, 1000, 1_000_000_000_000)
	if err := e.SubmitOrder(sell); err != nil {
		b.Fatal(err)
	}

	stopDrain := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			select {
			case <-e.Trades():
				// Drain trades.
			case <-stopDrain:
				return
			}
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buy := createBuyOrder(
			buyIDOffset+int64(i),
			1000,
			10,
		)

		if err := e.SubmitOrder(buy); err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	// Stop the engine while the trade consumer is still running.
	e.Stop()

	// Now the engine is completely stopped, so the consumer can exit safely.
	close(stopDrain)
	<-done
}

func BenchmarkEngineThroughput(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()

	e := engine.New("BTCUSDT", nil, nil)

	seed := createSellOrder(
		1,
		1000,
		1_000_000_000_000,
	)

	if err := e.SubmitOrder(seed); err != nil {
		b.Fatal(err)
	}

	// Continuously drain trades so the engine can never block
	// on the bounded trade queue.
	stopDrain := make(chan struct{})
	drainDone := make(chan struct{})

	go func() {
		defer close(drainDone)

		for {
			select {
			case <-e.Trades():
				// Drain trade.
			case <-stopDrain:
				return
			}
		}
	}()

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		buy := createBuyOrder(
			buyIDOffset+int64(i),
			1000,
			10,
		)

		if err := e.SubmitOrder(buy); err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	// Stop the engine while the trade consumer is still alive.
	e.Stop()

	close(stopDrain)
	<-drainDone

	// Report throughput directly.
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "orders/sec")
}

func BenchmarkEngineMultiProducer4(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()

	e := engine.New("BTCUSDT", nil, nil)

	seed := createSellOrder(
		1,
		1000,
		1_000_000_000_000,
	)

	if err := e.SubmitOrder(seed); err != nil {
		b.Fatal(err)
	}

	stopDrain := make(chan struct{})
	drainDone := make(chan struct{})

	go func() {
		defer close(drainDone)

		for {
			select {
			case <-e.Trades():
				// Drain trades.
			case <-stopDrain:
				return
			}
		}
	}()

	const producers = 4

	var wg sync.WaitGroup
	wg.Add(producers)

	b.StartTimer()

	for p := 0; p < producers; p++ {
		go func(producerID int) {
			defer wg.Done()

			for i := producerID; i < b.N; i += producers {
				buy := createBuyOrder(
					buyIDOffset+int64(i),
					1000,
					10,
				)

				if err := e.SubmitOrder(buy); err != nil {
					b.Error(err)
					return
				}
			}
		}(p)
	}

	wg.Wait()

	b.StopTimer()

	e.Stop()

	close(stopDrain)
	<-drainDone

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "orders/sec")
}

func BenchmarkEngineMultiProducer8(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()

	e := engine.New("BTCUSDT", nil, nil)

	seed := createSellOrder(
		1,
		1000,
		1_000_000_000_000,
	)

	if err := e.SubmitOrder(seed); err != nil {
		b.Fatal(err)
	}

	stopDrain := make(chan struct{})
	drainDone := make(chan struct{})

	go func() {
		defer close(drainDone)

		for {
			select {
			case <-e.Trades():
				// Drain trades.
			case <-stopDrain:
				return
			}
		}
	}()

	const producers = 8

	var wg sync.WaitGroup
	wg.Add(producers)

	b.StartTimer()

	for p := 0; p < producers; p++ {
		go func(producerID int) {
			defer wg.Done()

			for i := producerID; i < b.N; i += producers {
				buy := createBuyOrder(
					buyIDOffset+int64(i),
					1000,
					10,
				)

				if err := e.SubmitOrder(buy); err != nil {
					b.Error(err)
					return
				}
			}
		}(p)
	}

	wg.Wait()

	b.StopTimer()

	e.Stop()

	close(stopDrain)
	<-drainDone

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "orders/sec")
}

func BenchmarkEngineMultiFill(b *testing.B) {
	const levels = 100

	b.ReportAllocs()
	b.StopTimer()

	e := engine.New("BTCUSDT", nil, nil)

	b.Cleanup(func() {
		e.Stop()
	})

	buyID := int64(buyIDOffset)

	for i := 0; i < b.N; i++ {
		// Rebuild the 100-level book outside the timed section.
		for j := 0; j < levels; j++ {
			sell := createSellOrder(
				int64(i*levels+j+1),
				int64(1000+j),
				10,
			)

			sell.UserID = int64(200 + j)

			if err := e.SubmitOrder(sell); err != nil {
				b.Fatal(err)
			}
		}

		buy := createBuyOrder(
			buyID,
			1099,
			levels*10,
		)
		buyID++

		b.StartTimer()

		if err := e.SubmitOrder(buy); err != nil {
			b.Fatal(err)
		}

		for j := 0; j < levels; j++ {
			<-e.Trades()
		}

		b.StopTimer()

		b.ReportMetric(float64(levels), "trades/op")
	}
}

func BenchmarkEngineMixedWorkload(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()

	e := engine.New("BTCUSDT", nil, nil)

	// Large resting order at the best ask so limit/market buys
	// can execute without exhausting the liquidity.
	seed := createSellOrder(
		1,
		1000,
		1_000_000_000_000,
	)

	if err := e.SubmitOrder(seed); err != nil {
		b.Fatal(err)
	}

	// 15% of operations are cancel/modify operations.
	targetCount := (b.N / 20) * 3
	remainder := b.N % 20

	if remainder > 17 {
		targetCount++
	}
	if remainder > 18 {
		targetCount++
	}
	if remainder > 19 {
		targetCount++
	}

	// Create resting orders that will later be cancelled/modified.
	// All setup happens outside the timed section.
	for i := 0; i < targetCount; i++ {
		sell := createSellOrder(
			int64(i)+2,
			2000,
			10,
		)

		if err := e.SubmitOrder(sell); err != nil {
			b.Fatal(err)
		}
	}

	stopDrain := make(chan struct{})
	drainDone := make(chan struct{})

	go func() {
		defer close(drainDone)

		for {
			select {
			case <-e.Trades():
				// Drain trades.
			case <-stopDrain:
				return
			}
		}
	}()

	b.Cleanup(func() {
		// Keep the trade consumer alive while the engine shuts down.
		e.Stop()

		close(stopDrain)
		<-drainDone
	})

	targetIndex := 0
	buyID := int64(buyIDOffset)

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		switch i % 20 {
		// 70% LIMIT BUY
		case 0, 1, 2, 3, 4, 5, 6,
			7, 8, 9, 10, 11, 12, 13:

			buy := createBuyOrder(
				buyID,
				1000,
				10,
			)
			buyID++

			if err := e.SubmitOrder(buy); err != nil {
				b.Fatal(err)
			}

		// 15% MARKET BUY
		case 14, 15, 16:

			buy := createBuyOrder(
				buyID,
				0,
				10,
			)
			buy.Type = constants.OrderTypeMarket
			buy.Price = 0
			buyID++

			if err := e.SubmitOrder(buy); err != nil {
				b.Fatal(err)
			}

		// 10% CANCEL
		case 17, 18:

			orderID := int64(targetIndex) + 2
			targetIndex++

			if err := e.CancelOrder(orderID); err != nil {
				b.Fatal(err)
			}

		// 5% MODIFY
		case 19:

			orderID := int64(targetIndex) + 2
			targetIndex++

			if err := e.ModifyOrder(
				orderID,
				2100,
				5,
			); err != nil {
				b.Fatal(err)
			}
		}
	}

	b.StopTimer()

	b.ReportMetric(
		float64(b.N)/b.Elapsed().Seconds(),
		"orders/sec",
	)
}

type mixedOpKind uint8

const (
	mixedLimitBuy mixedOpKind = iota
	mixedMarketBuy
	mixedCancel
	mixedModify
)

type mixedOp struct {
	kind    mixedOpKind
	orderID int64
}

func BenchmarkEngineMixedWorkloadFixed(b *testing.B) {
	const (
		operations   = 100_000
		cancelCount  = operations / 10
		modifyCount  = operations / 20
		targetOrders = cancelCount + modifyCount
	)

	// Build the exact same workload once.
	ops := make([]mixedOp, 0, operations)

	cancelID := int64(2)
	modifyID := int64(2 + cancelCount)

	for i := 0; i < operations; i++ {
		switch i % 20 {
		// 70% limit buys
		case 0, 1, 2, 3, 4, 5, 6,
			7, 8, 9, 10, 11, 12, 13:
			ops = append(ops, mixedOp{
				kind: mixedLimitBuy,
			})

		// 15% market buys
		case 14, 15, 16:
			ops = append(ops, mixedOp{
				kind: mixedMarketBuy,
			})

		// 10% cancels
		case 17, 18:
			ops = append(ops, mixedOp{
				kind:    mixedCancel,
				orderID: cancelID,
			})
			cancelID++

		// 5% modifies
		case 19:
			ops = append(ops, mixedOp{
				kind:    mixedModify,
				orderID: modifyID,
			})
			modifyID++
		}
	}

	if len(ops) != operations {
		b.Fatalf("expected %d operations, got %d", operations, len(ops))
	}

	b.ReportAllocs()
	b.StopTimer()

	e := engine.New("BTCUSDT", nil, nil)

	// Large liquidity source for the BUY workloads.
	seed := createSellOrder(
		1,
		1000,
		1_000_000_000_000,
	)

	if err := e.SubmitOrder(seed); err != nil {
		b.Fatal(err)
	}

	// Resting orders used exclusively by cancel/modify operations.
	for i := 0; i < targetOrders; i++ {
		orderID := int64(i) + 2

		sell := createSellOrder(
			orderID,
			2000,
			10,
		)

		if err := e.SubmitOrder(sell); err != nil {
			b.Fatal(err)
		}
	}

	stopDrain := make(chan struct{})
	drainDone := make(chan struct{})

	go func() {
		defer close(drainDone)

		for {
			select {
			case <-e.Trades():
				// Drain trades.
			case <-stopDrain:
				return
			}
		}
	}()

	b.StartTimer()

	buyID := int64(buyIDOffset)

	for _, op := range ops {
		switch op.kind {
		case mixedLimitBuy:
			buy := createBuyOrder(
				buyID,
				1000,
				10,
			)
			buyID++

			if err := e.SubmitOrder(buy); err != nil {
				b.Fatal(err)
			}

		case mixedMarketBuy:
			buy := createBuyOrder(
				buyID,
				0,
				10,
			)
			buy.Type = constants.OrderTypeMarket
			buy.Price = 0
			buyID++

			if err := e.SubmitOrder(buy); err != nil {
				b.Fatal(err)
			}

		case mixedCancel:
			if err := e.CancelOrder(op.orderID); err != nil {
				b.Fatal(err)
			}

		case mixedModify:
			if err := e.ModifyOrder(
				op.orderID,
				2100,
				5,
			); err != nil {
				b.Fatal(err)
			}
		}
	}

	b.StopTimer()

	// Keep the trade consumer alive while the engine shuts down.
	e.Stop()

	close(stopDrain)
	<-drainDone

	b.ReportMetric(
		float64(operations),
		"orders/op",
	)

	b.ReportMetric(
		float64(operations)/b.Elapsed().Seconds(),
		"orders/sec",
	)
}

func BenchmarkEngineLimitOnly(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()

	e := engine.New("BTCUSDT", nil, nil)

	seed := createSellOrder(1, 1000, 1_000_000_000_000)
	if err := e.SubmitOrder(seed); err != nil {
		b.Fatal(err)
	}

	stopDrain := make(chan struct{})
	drainDone := make(chan struct{})

	go func() {
		defer close(drainDone)

		for {
			select {
			case <-e.Trades():
			case <-stopDrain:
				return
			}
		}
	}()

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		buy := createBuyOrder(
			buyIDOffset+int64(i),
			1000,
			10,
		)

		if err := e.SubmitOrder(buy); err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	e.Stop()
	close(stopDrain)
	<-drainDone
}

func BenchmarkEngineMarketOnly(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()

	e := engine.New("BTCUSDT", nil, nil)

	seed := createSellOrder(1, 1000, 1_000_000_000_000)
	if err := e.SubmitOrder(seed); err != nil {
		b.Fatal(err)
	}

	stopDrain := make(chan struct{})
	drainDone := make(chan struct{})

	go func() {
		defer close(drainDone)

		for {
			select {
			case <-e.Trades():
			case <-stopDrain:
				return
			}
		}
	}()

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		buy := createBuyOrder(
			buyIDOffset+int64(i),
			0,
			10,
		)
		buy.Type = constants.OrderTypeMarket
		buy.Price = 0

		if err := e.SubmitOrder(buy); err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	e.Stop()
	close(stopDrain)
	<-drainDone
}

func BenchmarkEngineCancelOnly(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()

	e := engine.New("BTCUSDT", nil, nil)

	for i := 0; i < b.N; i++ {
		sell := createSellOrder(
			int64(i)+2,
			2000,
			10,
		)

		if err := e.SubmitOrder(sell); err != nil {
			b.Fatal(err)
		}
	}

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		orderID := int64(i) + 2

		if err := e.CancelOrder(orderID); err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	e.Stop()
}

func BenchmarkEngineModifyOnly(b *testing.B) {
	const orders = 100_000

	b.ReportAllocs()
	b.StopTimer()

	e := engine.New("BTCUSDT", nil, nil)

	for i := 0; i < orders; i++ {
		sell := createSellOrder(
			int64(i)+2,
			2000,
			10,
		)

		if err := e.SubmitOrder(sell); err != nil {
			b.Fatal(err)
		}
	}

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		orderID := int64(i%orders) + 2

		if err := e.ModifyOrder(
			orderID,
			2100,
			5,
		); err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	e.Stop()
}
