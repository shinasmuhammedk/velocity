package matcher

import (
	"velocity/internal/domain/order"
	"velocity/internal/domain/trade"
	"velocity/internal/engine/orderbook"
	"velocity/pkg/constants"
	"velocity/pkg/helpers"
	"velocity/pkg/idgen"
	"velocity/pkg/timeutil"
)

type Matcher struct {
	book      *orderbook.OrderBook
	exhausted map[int64]bool
}

func New(book *orderbook.OrderBook) *Matcher {
	return &Matcher{
		book:      book,
		exhausted: make(map[int64]bool),
	}
}

func (m *Matcher) Match(order *order.Order) ([]*trade.Trade, error) {

	if m.isPostOnlyCrossing(order) {
		order.Status = constants.OrderStatusRejected
		return nil, nil
	}

	if order.TimeInForce == constants.TimeInForceFOK {

		if order.Side == constants.OrderSideBuy && !m.canFullyFill(order) {

			order.Status = constants.OrderStatusCancelled
			return nil, nil
		}

		if order.Side == constants.OrderSideSell && !m.canFullyFill(order) {

			order.Status = constants.OrderStatusCancelled
			return nil, nil
		}
	}

	if order.Side == constants.OrderSideBuy {
		return m.matchBuyOrder(order), nil
	}

	return m.matchSellOrder(order), nil
}

func (m *Matcher) matchBuyOrder(incoming *order.Order) []*trade.Trade {

	var trades []*trade.Trade
	clear(m.exhausted)

	for incoming.Remaining > 0 {
		bestAsk := m.book.BestAskExcluding(m.exhausted)

		if bestAsk == nil {
			break
		}

		if incoming.Type != constants.OrderTypeMarket && bestAsk.Price > incoming.Price {
			break
		}

		resting := bestAsk.FindFirstExcludingUser(incoming.UserID)

		if resting == nil {
			m.exhausted[bestAsk.Price] = true
			continue
		}

		tradeQty := helpers.MinInt64(
			incoming.Remaining,
			resting.Remaining,
		)


		t := &trade.Trade{
			ID: idgen.UUID(),

			BuyOrderID:  incoming.ID,
			SellOrderID: resting.ID,

			BuyerID:  incoming.UserID,
			SellerID: resting.UserID,

			Symbol: incoming.Symbol,

			Price: bestAsk.Price,

			Quantity: tradeQty,

			ExecutedAt: timeutil.UTCNow(),
		}


		trades = append(trades, t)

		incoming.Remaining -= tradeQty
		resting.Remaining -= tradeQty

		incoming.Filled += tradeQty
		resting.Filled += tradeQty

		if resting.Remaining == 0 {
			resting.Status = constants.OrderStatusFilled

			bestAsk.Remove(resting)

			m.book.RemoveOrderIndex(resting.ID)
		} else {
			resting.Status = constants.OrderStatusPartiallyFilled
		}

		if bestAsk.IsEmpty() {
			m.book.RemoveAskLevel(bestAsk.Price)
		}

	}

	if incoming.Remaining == 0 {
		incoming.Status = constants.OrderStatusFilled

	} else if incoming.Remaining < incoming.Quantity {
		incoming.Status = constants.OrderStatusPartiallyFilled

	} else if incoming.TimeInForce == constants.TimeInForceIOC {
		incoming.Status = constants.OrderStatusCancelled

	} else {
		incoming.Status = constants.OrderStatusOpen
	}

	if incoming.Remaining > 0 &&
		incoming.Type != constants.OrderTypeMarket &&
		(incoming.TimeInForce == constants.TimeInForceGTC ||
			incoming.TimeInForce == constants.TimeInForcePostOnly) {

		m.book.AddOrder(incoming)
	}

	return trades
}

func (m *Matcher) matchSellOrder(incoming *order.Order) []*trade.Trade {

	var trades []*trade.Trade
	clear(m.exhausted)

	for incoming.Remaining > 0 {

		bestBid := m.book.BestBidExcluding(m.exhausted)

		if bestBid == nil {
			break
		}

		// Seller won't sell below their limit price
		if incoming.Type != constants.OrderTypeMarket && bestBid.Price < incoming.Price {
			break
		}

		resting := bestBid.FindFirstExcludingUser(incoming.UserID)

		if resting == nil {
			m.exhausted[bestBid.Price] = true
			continue
		}

		tradeQty := helpers.MinInt64(
			incoming.Remaining,
			resting.Remaining,
		)

		t := &trade.Trade{
			ID: idgen.UUID(),

			BuyOrderID:  resting.ID,
			SellOrderID: incoming.ID,

			BuyerID:  resting.UserID,
			SellerID: incoming.UserID,

			Symbol: incoming.Symbol,

			// Maker price rule
			Price: bestBid.Price,

			Quantity: tradeQty,

			ExecutedAt: timeutil.UTCNow(),
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

			bestBid.Remove(resting)

			m.book.RemoveOrderIndex(resting.ID)
		} else {
			resting.Status = constants.OrderStatusPartiallyFilled
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

	} else if incoming.TimeInForce == constants.TimeInForceIOC {
		incoming.Status = constants.OrderStatusCancelled

	} else {
		incoming.Status = constants.OrderStatusOpen
	}

	// Add remaining quantity back to book
	if incoming.Remaining > 0 &&
		incoming.Type != constants.OrderTypeMarket &&
		(incoming.TimeInForce == constants.TimeInForceGTC ||
			incoming.TimeInForce == constants.TimeInForcePostOnly) {

		m.book.AddOrder(incoming)
	}

	return trades
}

func (m *Matcher) canFullyFill(incoming *order.Order) bool {
	var available int64

	clear(m.exhausted)

	if incoming.Side == constants.OrderSideBuy {
		for {
			level := m.book.BestAskExcluding(m.exhausted)

			if level == nil {
				break
			}

			// Respect limit price
			if incoming.Type != constants.OrderTypeMarket &&
				level.Price > incoming.Price {
				break
			}

			for e := level.Orders.Front(); e != nil; e = e.Next() {
				o := e.Value.(*order.Order)

				if o.UserID == incoming.UserID {
					continue
				}

				available += o.Remaining

				if available >= incoming.Quantity {
					return true
				}
			}

			m.exhausted[level.Price] = true

		}
	} else {
		for {
			level := m.book.BestBidExcluding(m.exhausted)

			if level == nil {
				break
			}

			if incoming.Type != constants.OrderTypeMarket &&
				level.Price < incoming.Price {
				break
			}

			for e := level.Orders.Front(); e != nil; e = e.Next() {
				o := e.Value.(*order.Order)

				if o.UserID == incoming.UserID {
					continue
				}

				available += o.Remaining

				if available >= incoming.Quantity {
					return true
				}
			}

			m.exhausted[level.Price] = true
		}
	}

	return false
}

func (m *Matcher) isPostOnlyCrossing(incoming *order.Order) bool {
	if incoming.TimeInForce != constants.TimeInForcePostOnly {
		return false
	}

	clear(m.exhausted)

	if incoming.Side == constants.OrderSideBuy {
		for {
			level := m.book.BestAskExcluding(m.exhausted)
			if level == nil {
				return false
			}
			if level.Price > incoming.Price {
				return false // no longer crosses — nothing behind this can cross either
			}

			resting := level.FindFirstExcludingUser(incoming.UserID)

			if resting != nil {
				return true // a real counterparty exists — this would actually trade
			}

			m.exhausted[level.Price] = true // this level is entirely self-orders, check the next one
		}
	}

	for {
		level := m.book.BestBidExcluding(m.exhausted)
		if level == nil {
			return false
		}
		if level.Price < incoming.Price {
			return false
		}

		resting := level.FindFirstExcludingUser(incoming.UserID)

		if resting != nil {
			return true
		}

		m.exhausted[level.Price] = true
	}
}
