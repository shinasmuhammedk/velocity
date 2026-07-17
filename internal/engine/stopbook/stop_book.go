package stopbook

import (
	"sync"
	"velocity/internal/domain/order"
	"velocity/pkg/constants"
	apperr "velocity/pkg/errors"
)

type StopBook struct {
	buyStops  map[int64][]*order.Order
	sellStops map[int64][]*order.Order

	orderIndex map[string]*order.Order
	mu         sync.Mutex
}

func New() *StopBook {
	return &StopBook{
		buyStops:   make(map[int64][]*order.Order),
		sellStops:  make(map[int64][]*order.Order),
		orderIndex: make(map[string]*order.Order),
	}
}

func (s *StopBook) Add(o *order.Order) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if o.Side == constants.OrderSideBuy {
		s.buyStops[o.StopPrice] =
			append(s.buyStops[o.StopPrice], o)
	} else {
		s.sellStops[o.StopPrice] =
			append(s.sellStops[o.StopPrice], o)
	}

	s.orderIndex[o.ID] = o
}

func (s *StopBook) Trigger(price int64) []*order.Order {
	s.mu.Lock()
	defer s.mu.Unlock()

	var triggered []*order.Order

	for stopPrice, orders := range s.buyStops {
		if price >= stopPrice {
			triggered = append(triggered, orders...)

			for _, o := range orders {
				delete(s.orderIndex, o.ID)
			}

			delete(s.buyStops, stopPrice)
		}
	}

	for stopPrice, orders := range s.sellStops {
		if price <= stopPrice {
			triggered = append(triggered, orders...)

			for _, o := range orders {
				delete(s.orderIndex, o.ID)
			}

			delete(s.sellStops, stopPrice)
		}
	}

	return triggered
}

func (s *StopBook) CancelOrder(orderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	o, ok := s.orderIndex[orderID]
	if !ok {
		return apperr.ErrStopOrderNotFound
	}

	var levels map[int64][]*order.Order

	if o.Side == constants.OrderSideBuy {
		levels = s.buyStops
	} else {
		levels = s.sellStops
	}

	orders := levels[o.StopPrice]

	for i, existing := range orders {
		if existing.ID == orderID {
			orders = append(
				orders[:i],
				orders[i+1:]...,
			)

			if len(orders) == 0 {
				delete(levels, o.StopPrice)
			} else {
				levels[o.StopPrice] = orders
			}

			o.Status = constants.OrderStatusCancelled

			delete(s.orderIndex, orderID)

			return nil
		}
	}

	return apperr.ErrStopOrderNotFound
}

func (s *StopBook) Orders() []*order.Order {
	s.mu.Lock()
	defer s.mu.Unlock()

	orders := make(
		[]*order.Order,
		0,
		len(s.orderIndex),
	)

	// Buy stops
	for _, stopOrders := range s.buyStops {
		orders = append(
			orders,
			stopOrders...,
		)
	}

	// Sell stops
	for _, stopOrders := range s.sellStops {
		orders = append(
			orders,
			stopOrders...,
		)
	}

	return orders
}