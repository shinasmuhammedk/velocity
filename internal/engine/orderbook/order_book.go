package orderbook

import (
	"container/heap"
	"sync"

	"velocity/internal/domain/order"
	"velocity/internal/engine/pricelevel"
	"velocity/pkg/constants"
	"velocity/pkg/errors"
	"velocity/pkg/timeutil"
)

// askHeap is a min-heap of prices — lowest price is always at the root,
// since the best ask (for a buyer) is the cheapest seller.
type askHeap []int64

func (h askHeap) Len() int           { return len(h) }
func (h askHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h askHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *askHeap) Push(x any)        { *h = append(*h, x.(int64)) }
func (h *askHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

// bidHeap is a max-heap of prices — highest price is always at the root,
// since the best bid (for a seller) is the buyer willing to pay the most.
type bidHeap []int64

func (h bidHeap) Len() int           { return len(h) }
func (h bidHeap) Less(i, j int) bool { return h[i] > h[j] } // reversed for max-heap
func (h bidHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *bidHeap) Push(x any)        { *h = append(*h, x.(int64)) }
func (h *bidHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

type OrderLocation struct {
	Order *order.Order
	Level *pricelevel.PriceLevel
}

type OrderBook struct {
	Symbol string

	Bids map[int64]*pricelevel.PriceLevel
	Asks map[int64]*pricelevel.PriceLevel

	Orders map[string]*OrderLocation

	bidPrices bidHeap
	askPrices askHeap

	mu sync.RWMutex
}

func New(symbol string) *OrderBook {
	return &OrderBook{
		Symbol:    symbol,
		Bids:      make(map[int64]*pricelevel.PriceLevel),
		Asks:      make(map[int64]*pricelevel.PriceLevel),
		Orders:    make(map[string]*OrderLocation),
		bidPrices: bidHeap{},
		askPrices: askHeap{},
	}
}

func (b *OrderBook) AddOrder(o *order.Order) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.addOrderWithoutLock(o)
}

// BestBid returns the highest bid price level in O(1) — the heap root.
func (b *OrderBook) BestBid() *pricelevel.PriceLevel {
	b.mu.Lock() // Lock (not RLock) — may need to pop stale entries below
	defer b.mu.Unlock()

	return b.peekBid(nil)
}

// BestAsk returns the lowest ask price level in O(1) — the heap root.
func (b *OrderBook) BestAsk() *pricelevel.PriceLevel {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.peekAsk(nil)
}

// BestBidExcluding returns the highest bid price level not in excluded.
func (b *OrderBook) BestBidExcluding(excluded map[int64]bool) *pricelevel.PriceLevel {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.peekBid(excluded)
}

// BestAskExcluding returns the lowest ask price level not in excluded.
func (b *OrderBook) BestAskExcluding(excluded map[int64]bool) *pricelevel.PriceLevel {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.peekAsk(excluded)
}

// peekBid walks the heap root-first, discarding stale entries (price levels
// already removed) permanently, and skipping-but-preserving excluded ones.
func (b *OrderBook) peekBid(excluded map[int64]bool) *pricelevel.PriceLevel {
	var skipped []int64
	var result *pricelevel.PriceLevel

	for b.bidPrices.Len() > 0 {
		price := b.bidPrices[0]

		level, exists := b.Bids[price]
		if !exists {
			heap.Pop(&b.bidPrices) // stale — level was removed, discard for good
			continue
		}

		if excluded != nil && excluded[price] {
			heap.Pop(&b.bidPrices)
			skipped = append(skipped, price)
			continue
		}

		result = level
		break
	}

	for _, p := range skipped {
		heap.Push(&b.bidPrices, p) // put back — still valid for other orders
	}

	return result
}

func (b *OrderBook) peekAsk(excluded map[int64]bool) *pricelevel.PriceLevel {
	var skipped []int64
	var result *pricelevel.PriceLevel

	for b.askPrices.Len() > 0 {
		price := b.askPrices[0]

		level, exists := b.Asks[price]
		if !exists {
			heap.Pop(&b.askPrices)
			continue
		}

		if excluded != nil && excluded[price] {
			heap.Pop(&b.askPrices)
			skipped = append(skipped, price)
			continue
		}

		result = level
		break
	}

	for _, p := range skipped {
		heap.Push(&b.askPrices, p)
	}

	return result
}

func (b *OrderBook) RemoveBidLevel(price int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.Bids, price) // heap entry becomes "stale" — cleaned up lazily on next peek
}

func (b *OrderBook) RemoveAskLevel(price int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.Asks, price)
}

func (b *OrderBook) CancelOrder(orderID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	location, exists := b.Orders[orderID]
	if !exists {
		return errors.ErrOrderNotFound
	}

	o := location.Order
	level := location.Level

	if o.Status == constants.OrderStatusFilled {
		return errors.ErrOrderFilled
	}

	if o.Status == constants.OrderStatusCancelled {
		return errors.ErrOrderCancelled
	}

	// Remove order from the FIFO queue
	level.Remove(o)

	// Update order state
	o.Status = constants.OrderStatusCancelled
	o.UpdatedAt = timeutil.UTCNow()

	// Remove empty price level
	if level.IsEmpty() {
		if o.Side == constants.OrderSideBuy {
			delete(b.Bids, level.Price)
		} else {
			delete(b.Asks, level.Price)
		}
	}

	// Remove from order index
	delete(b.Orders, orderID)

	return nil
}

func (b *OrderBook) RemoveOrderIndex(orderID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.Orders, orderID)
}

func (b *OrderBook) ModifyOrder(orderID string, newPrice int64, newQuantity int64) error {

	b.mu.Lock()
	defer b.mu.Unlock()

	location, exists := b.Orders[orderID]
	if !exists {
		return errors.ErrOrderNotFound
	}

	o := location.Order
	level := location.Level

	// Cannot modify completed orders
	if o.Status == constants.OrderStatusFilled {
		return errors.ErrOrderFilled
	}

	if o.Status == constants.OrderStatusCancelled {
		return errors.ErrOrderCancelled
	}

	priceChanged := newPrice != o.Price
	qtyIncreased := newQuantity > o.Quantity

	if newQuantity < o.Filled {
		return errors.ErrInvalidOrder
	}

	// Simple case:
	// quantity reduction keeps priority
	if !priceChanged && !qtyIncreased {

		o.Quantity = newQuantity
		o.Remaining = newQuantity - o.Filled
		o.UpdatedAt = timeutil.UTCNow()

		return nil
	}

	// Price change or quantity increase loses priority
	level.Remove(o)

	if level.IsEmpty() {
		if o.Side == constants.OrderSideBuy {
			delete(b.Bids, level.Price)
		} else {
			delete(b.Asks, level.Price)
		}
	}

	o.Price = newPrice
	o.Quantity = newQuantity
	o.Remaining = newQuantity - o.Filled
	o.UpdatedAt = timeutil.UTCNow()

	delete(b.Orders, orderID)

	b.addOrderWithoutLock(o)

	return nil
}

func (b *OrderBook) addOrderWithoutLock(o *order.Order) {
	var level *pricelevel.PriceLevel

	if o.Side == constants.OrderSideBuy {
		var exists bool
		level, exists = b.Bids[o.Price]

		if !exists {
			level = pricelevel.New(o.Price)
			b.Bids[o.Price] = level
			heap.Push(&b.bidPrices, o.Price)
		}
	} else {
		var exists bool
		level, exists = b.Asks[o.Price]

		if !exists {
			level = pricelevel.New(o.Price)
			b.Asks[o.Price] = level
			heap.Push(&b.askPrices, o.Price)
		}
	}

	level.AddOrder(o)

	b.Orders[o.ID] = &OrderLocation{
		Order: o,
		Level: level,
	}
}


// RemoveFilledOrder removes a fully-filled order from both its price level
// and the order index. Called by the matcher after a resting order is
// completely consumed — keeps the index in sync with what matcher.go
// already does directly on the PriceLevel.
func (b *OrderBook) RemoveFilledOrder(o *order.Order) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.Orders, o.ID)
}