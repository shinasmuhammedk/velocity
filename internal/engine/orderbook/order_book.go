package orderbook

import (
	"sync"

	"velocity/internal/domain/depth"
	"velocity/internal/domain/order"
	"velocity/internal/engine/pricelevel"
	"velocity/pkg/constants"
	"velocity/pkg/errors"
	"velocity/pkg/timeutil"

	"github.com/google/btree"
)

// priceTreeDegree controls the branching factor of the price trees.
// A higher degree means fewer, wider nodes: more comparisons per node
// but fewer node visits (and fewer cache-unfriendly pointer hops) to
// reach a given depth. 32 is the value used in google/btree's own
// benchmarks and documentation as a good general-purpose default; the
// price trees here are read far more often (every match attempt) than
// written (only on a level's first order or its last cancellation), so
// biasing toward fewer levels of indirection is the right trade-off.
const priceTreeDegree = 32

// newBidTree returns a price tree ordered so that Min() yields the
// highest price. Bids are "best first" from the buyer's perspective:
// the buyer willing to pay the most is served first. Flipping the
// comparator (rather than negating prices or maintaining a separate
// max-tracking structure) means every other tree operation — Ascend,
// Delete, range queries — automatically inherits "best bid first"
// semantics for free.
func newBidTree() *btree.BTreeG[int64] {
	return btree.NewG[int64](priceTreeDegree, func(a, b int64) bool {
		return a > b
	})
}

// newAskTree returns a price tree in natural ascending order, so
// Min() yields the lowest price — the cheapest seller, who is served
// first.
func newAskTree() *btree.BTreeG[int64] {
	return btree.NewG[int64](priceTreeDegree, func(a, b int64) bool {
		return a < b
	})
}

type OrderLocation struct {
	Order *order.Order
	Level *pricelevel.PriceLevel
}

// OrderBook holds one symbol's resting orders.
//
// Bids and Asks give O(1) lookup of the PriceLevel at an exact price.
// bidPrices and askPrices are ordered trees over the same set of
// prices, giving O(log n) best-price lookup, O(log n) insertion and
// removal, and O(k + log n) ordered iteration for the first k levels —
// where a plain slice-and-sort approach would cost O(n log n) on every
// call regardless of how many levels are actually needed.
//
// The map and the corresponding tree are always kept in lockstep: a
// price is inserted into both when its first order arrives and deleted
// from both the moment its last order leaves. There is deliberately no
// lazy deletion here — every removal path (CancelOrder, ModifyOrder,
// RemoveBidLevel, RemoveAskLevel) deletes from the tree in the same
// critical section as the map, so the tree never carries a stale price
// forward for another caller to trip over.
type OrderBook struct {
	Symbol string

	Bids map[int64]*pricelevel.PriceLevel
	Asks map[int64]*pricelevel.PriceLevel

	Orders map[int64]*OrderLocation

	bidPrices *btree.BTreeG[int64]
	askPrices *btree.BTreeG[int64]

	mu sync.RWMutex
}

func New(symbol string) *OrderBook {
	return &OrderBook{
		Symbol:    symbol,
		Bids:      make(map[int64]*pricelevel.PriceLevel),
		Asks:      make(map[int64]*pricelevel.PriceLevel),
		Orders:    make(map[int64]*OrderLocation),
		bidPrices: newBidTree(),
		askPrices: newAskTree(),
	}
}

// SideCounts returns the number of resting orders on each side.
//
// Used by the engine's metrics sampler. It walks the order index rather
// than the price-level trees so the cost is a single O(n) pass under a
// read lock with no allocation, rather than materialising depth levels.
func (b *OrderBook) SideCounts() (bids int, asks int) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, loc := range b.Orders {
		if loc == nil || loc.Order == nil {
			continue
		}

		if loc.Order.Side == constants.OrderSideBuy {
			bids++
		} else {
			asks++
		}
	}

	return bids, asks
}

func (b *OrderBook) AddOrder(o *order.Order) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.addOrderWithoutLock(o)
}

// BestBid returns the highest bid price level in O(log n).
func (b *OrderBook) BestBid() *pricelevel.PriceLevel {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.peekBid(nil)
}

// BestAsk returns the lowest ask price level in O(log n).
func (b *OrderBook) BestAsk() *pricelevel.PriceLevel {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.peekAsk(nil)
}

// BestBidExcluding returns the highest bid price level not in excluded.
func (b *OrderBook) BestBidExcluding(excluded map[int64]bool) *pricelevel.PriceLevel {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.peekBid(excluded)
}

// BestAskExcluding returns the lowest ask price level not in excluded.
func (b *OrderBook) BestAskExcluding(excluded map[int64]bool) *pricelevel.PriceLevel {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.peekAsk(excluded)
}

// peekBid walks the bid tree best-price-first, in O(log n + k) where k
// is the number of consecutive excluded prices at the front — typically
// zero or one, since exclusion here exists only for self-trade
// prevention against a single counterparty. Because the tree is never
// allowed to hold a stale price (see OrderBook's doc comment), every
// price this walk visits is guaranteed to have a live level in Bids.
func (b *OrderBook) peekBid(excluded map[int64]bool) *pricelevel.PriceLevel {
	var result *pricelevel.PriceLevel

	b.bidPrices.Ascend(func(price int64) bool {
		if excluded != nil && excluded[price] {
			return true // keep walking past an excluded price
		}

		result = b.Bids[price]

		return false // stop — first non-excluded price is the answer
	})

	return result
}

// peekAsk is peekBid's mirror image for the ask side.
func (b *OrderBook) peekAsk(excluded map[int64]bool) *pricelevel.PriceLevel {
	var result *pricelevel.PriceLevel

	b.askPrices.Ascend(func(price int64) bool {
		if excluded != nil && excluded[price] {
			return true
		}

		result = b.Asks[price]

		return false
	})

	return result
}

// RemoveBidLevel deletes a bid price level from both the map and the
// tree in one step, so no caller can ever observe the tree still
// carrying a price that no longer has a level behind it.
func (b *OrderBook) RemoveBidLevel(price int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.Bids, price)
	b.bidPrices.Delete(price)
}

// RemoveAskLevel is RemoveBidLevel's mirror image for the ask side.
func (b *OrderBook) RemoveAskLevel(price int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.Asks, price)
	b.askPrices.Delete(price)
}

func (b *OrderBook) CancelOrder(orderID int64) (*order.Order, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	location, exists := b.Orders[orderID]
	if !exists {
		return nil, errors.ErrOrderNotFound
	}

	o := location.Order
	level := location.Level

	if o.Status == constants.OrderStatusFilled {
		return nil, errors.ErrOrderFilled
	}

	if o.Status == constants.OrderStatusCancelled {
		return nil, errors.ErrOrderCancelled
	}

	// Remove order from the FIFO queue
	level.Remove(o)

	// Update order state
	o.Status = constants.OrderStatusCancelled
	o.UpdatedAt = timeutil.UTCNow()

	// Remove empty price level from both the map and the tree together,
	// so the tree never outlives the level it points at.
	if level.IsEmpty() {
		if o.Side == constants.OrderSideBuy {
			delete(b.Bids, level.Price)
			b.bidPrices.Delete(level.Price)
		} else {
			delete(b.Asks, level.Price)
			b.askPrices.Delete(level.Price)
		}
	}

	// Remove from order index
	delete(b.Orders, orderID)

	return o, nil
}

func (b *OrderBook) RemoveOrderIndex(orderID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.Orders, orderID)
}

func (b *OrderBook) ModifyOrder(orderID int64, newPrice int64, newQuantity int64) error {

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
			b.bidPrices.Delete(level.Price)
		} else {
			delete(b.Asks, level.Price)
			b.askPrices.Delete(level.Price)
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
			b.bidPrices.ReplaceOrInsert(o.Price)
		}
	} else {
		var exists bool
		level, exists = b.Asks[o.Price]

		if !exists {
			level = pricelevel.New(o.Price)
			b.Asks[o.Price] = level
			b.askPrices.ReplaceOrInsert(o.Price)
		}
	}

	level.AddOrder(o)

	b.Orders[o.ID] = &OrderLocation{
		Order: o,
		Level: level,
	}
}

func (b *OrderBook) GetOrder(orderID int64) *order.Order {
	b.mu.RLock()
	defer b.mu.RUnlock()

	location, exists := b.Orders[orderID]
	if !exists {
		return nil
	}
	return location.Order
}

// BidLevels returns up to limit bid levels, highest price first.
//
// Walking the tree in its own order (best-first, by construction — see
// newBidTree) and stopping once limit levels are collected costs
// O(limit + log n): the log n gets us to the first entry, and each
// further entry is amortised O(1). The map-and-sort approach this
// replaced cost O(n log n) on every single call regardless of limit,
// since it had to collect and sort every price in the book before it
// could take the top few.
func (b *OrderBook) BidLevels(limit int) []depth.Level {
	b.mu.RLock()
	defer b.mu.RUnlock()

	levels := make([]depth.Level, 0, limit)

	b.bidPrices.Ascend(func(price int64) bool {
		if len(levels) >= limit {
			return false
		}

		level, exists := b.Bids[price]
		if !exists {
			return true
		}

		var qty int64

		for e := level.Orders.Front(); e != nil; e = e.Next() {
			o := e.Value.(*order.Order)
			qty += o.Remaining
		}

		levels = append(levels, depth.Level{
			Price:    price,
			Quantity: qty,
		})

		return true
	})

	return levels
}

// AskLevels is BidLevels' mirror image for the ask side.
func (b *OrderBook) AskLevels(limit int) []depth.Level {
	b.mu.RLock()
	defer b.mu.RUnlock()

	levels := make([]depth.Level, 0, limit)

	b.askPrices.Ascend(func(price int64) bool {
		if len(levels) >= limit {
			return false
		}

		level, exists := b.Asks[price]
		if !exists {
			return true
		}

		var qty int64

		for e := level.Orders.Front(); e != nil; e = e.Next() {
			o := e.Value.(*order.Order)
			qty += o.Remaining
		}

		levels = append(levels, depth.Level{
			Price:    price,
			Quantity: qty,
		})

		return true
	})

	return levels
}

func (b *OrderBook) BestBidPrice() int64 {
	if level := b.BestBid(); level != nil {
		return level.Price
	}

	return 0
}

func (b *OrderBook) BestAskPrice() int64 {
	if level := b.BestAsk(); level != nil {
		return level.Price
	}

	return 0
}

// ActiveOrders returns every resting order, bids first (best price
// first), then asks (best price first) — the same ordering the old
// sort-based implementation produced, now via an O(n) tree walk instead
// of an O(n log n) sort of every price on each call.
func (b *OrderBook) ActiveOrders() []*order.Order {
	b.mu.RLock()
	defer b.mu.RUnlock()

	orders := make([]*order.Order, 0, len(b.Orders))

	// -----------------------------
	// Bids (highest price first)
	// -----------------------------
	b.bidPrices.Ascend(func(price int64) bool {
		level, exists := b.Bids[price]
		if !exists {
			return true
		}

		for e := level.Orders.Front(); e != nil; e = e.Next() {
			orders = append(orders, e.Value.(*order.Order))
		}

		return true
	})

	// -----------------------------
	// Asks (lowest price first)
	// -----------------------------
	b.askPrices.Ascend(func(price int64) bool {
		level, exists := b.Asks[price]
		if !exists {
			return true
		}

		for e := level.Orders.Front(); e != nil; e = e.Next() {
			orders = append(orders, e.Value.(*order.Order))
		}

		return true
	})

	return orders
}