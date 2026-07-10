package pricelevel

import (
	"container/list"

	"velocity/internal/domain/order"
)

type PriceLevel struct {
	Price int64

	// FIFO queue for price-time priority
	Orders *list.List
}

func New(price int64) *PriceLevel {
	return &PriceLevel{
		Price:  price,
		Orders: list.New(),
	}
}

func (p *PriceLevel) AddOrder(o *order.Order) {
	p.Orders.PushBack(o)
}

func (p *PriceLevel) Front() *order.Order {
	if p.Orders.Len() == 0 {
		return nil
	}

	return p.Orders.Front().Value.(*order.Order)
}

func (p *PriceLevel) RemoveFront() {
	if p.Orders.Len() == 0 {
		return
	}

	p.Orders.Remove(p.Orders.Front())
}

func (p *PriceLevel) IsEmpty() bool {
	return p.Orders.Len() == 0
}

func (p *PriceLevel) Size() int {
	return p.Orders.Len()
}

// FindFirst walks the queue front-to-back and returns the first order that
// satisfies the given predicate, skipping any that don't. Returns nil if
// no order matches (e.g. every resting order here belongs to the same
// user as the incoming order, so nothing is eligible to match).
func (p *PriceLevel) FindFirst(match func(o *order.Order) bool) *order.Order {
	for e := p.Orders.Front(); e != nil; e = e.Next() {
		o := e.Value.(*order.Order)
		if match(o) {
			return o
		}
	}
	return nil
}

// Remove deletes a specific order from the queue, wherever it is in line —
// not just the front. Needed because self-trade prevention may need to
// remove an order that isn't at the very front.
func (p *PriceLevel) Remove(target *order.Order) {
	for e := p.Orders.Front(); e != nil; e = e.Next() {
		o := e.Value.(*order.Order)
		if o.ID == target.ID {
			p.Orders.Remove(e)
			return
		}
	}
}