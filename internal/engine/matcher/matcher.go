package matcher

import (
	"time"
	"velocity/internal/domain/order"
	"velocity/internal/domain/trade"
	"velocity/internal/engine/orderbook"
	"velocity/pkg/constants"
	"velocity/pkg/helpers"

	"github.com/google/uuid"
)

type Matcher struct {
	book *orderbook.OrderBook
}

func New(book *orderbook.OrderBook) *Matcher {
	return &Matcher{
		book: book,
	}
}

func (m *Matcher) Match(
	order *order.Order,
) ([]*trade.Trade, error) {

	if order.Side == constants.OrderSideBuy {
		return m.matchBuyOrder(order), nil
	}

	return m.matchSellOrder(order), nil

}

func (m *Matcher) matchBuyOrder(
	incoming *order.Order,
) []*trade.Trade {

	var trades []*trade.Trade
	exhausted := make(map[int64]bool)

	for incoming.Remaining > 0 {
		bestAsk := m.book.BestAskExcluding(exhausted)

		if bestAsk == nil {
			break
		}

		if incoming.Type != constants.OrderTypeMarket && bestAsk.Price > incoming.Price {
			break
		}

		resting := bestAsk.FindFirst(func(o *order.Order) bool {
			return o.UserID != incoming.UserID
		})

		if resting == nil {
			exhausted[bestAsk.Price] = true
			continue
		}

		tradeQty := helpers.MinInt64(
			incoming.Remaining,
			resting.Remaining,
		)

		t := &trade.Trade{
			ID: uuid.New().String(),

			BuyOrderID:  incoming.ID,
			SellOrderID: resting.ID,

			BuyerID:  incoming.UserID,
			SellerID: resting.UserID,

			Symbol: incoming.Symbol,

			Price: bestAsk.Price,

			Quantity: tradeQty,

			ExecutedAt: time.Now().UTC(),
		}

		trades = append(trades, t)

		incoming.Remaining -= tradeQty
		resting.Remaining -= tradeQty

		incoming.Filled += tradeQty
		resting.Filled += tradeQty

		if resting.Remaining == 0 {
			resting.Status = constants.OrderStatusFilled
		} else {
			resting.Status = constants.OrderStatusPartiallyFilled
		}

		if resting.Remaining == 0 {
			bestAsk.Remove(resting)
		}

		if bestAsk.IsEmpty() {
			m.book.RemoveAskLevel(bestAsk.Price)
		}

	}

	if incoming.Remaining == 0 {
		incoming.Status = constants.OrderStatusFilled
	} else if incoming.Remaining < incoming.Quantity {
		incoming.Status = constants.OrderStatusPartiallyFilled
	} else {
		incoming.Status = constants.OrderStatusOpen
	}

	if incoming.Remaining > 0 && incoming.Type != constants.OrderTypeMarket {
		m.book.AddOrder(incoming)
	}

	return trades
}

func (m *Matcher) matchSellOrder(
	incoming *order.Order,
) []*trade.Trade {

	var trades []*trade.Trade
	exhausted := make(map[int64]bool)

	for incoming.Remaining > 0 {

		bestBid := m.book.BestBidExcluding(exhausted)

		if bestBid == nil {
			break
		}

		// Seller won't sell below their limit price
		if incoming.Type != constants.OrderTypeMarket && bestBid.Price < incoming.Price {
			break
		}

		resting := bestBid.FindFirst(func(o *order.Order) bool {
			return o.UserID != incoming.UserID
		})

		if resting == nil {
			exhausted[bestBid.Price] = true
			continue
		}

		tradeQty := helpers.MinInt64(
			incoming.Remaining,
			resting.Remaining,
		)

		t := &trade.Trade{
			ID: uuid.New().String(),

			BuyOrderID:  resting.ID,
			SellOrderID: incoming.ID,

			BuyerID:  resting.UserID,
			SellerID: incoming.UserID,

			Symbol: incoming.Symbol,

			// Maker price rule
			Price: bestBid.Price,

			Quantity: tradeQty,

			ExecutedAt: time.Now().UTC(),
		}

		trades = append(trades, t)

		// Update quantities
		incoming.Remaining -= tradeQty
		resting.Remaining -= tradeQty

		incoming.Filled += tradeQty
		resting.Filled += tradeQty

		// Update resting order status
		if resting.Remaining == 0 {
			resting.Status = constants.OrderStatusFilled
		} else {
			resting.Status = constants.OrderStatusPartiallyFilled
		}

		// Remove fully filled resting order
		if resting.Remaining == 0 {
			bestBid.Remove(resting)
		}

		// Remove empty price level
		if bestBid.IsEmpty() {
			m.book.RemoveBidLevel(bestBid.Price)
		}
	}

	// Update incoming order status
	if incoming.Remaining == 0 {
		incoming.Status = constants.OrderStatusFilled
	} else if incoming.Remaining < incoming.Quantity {
		incoming.Status = constants.OrderStatusPartiallyFilled
	} else {
		incoming.Status = constants.OrderStatusOpen
	}

	// Add remaining quantity back to book
	if incoming.Remaining > 0 && incoming.Type != constants.OrderTypeMarket {
		m.book.AddOrder(incoming)
	}

	return trades
}
