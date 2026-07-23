package userstream

import (
	"fmt"

	"velocity/internal/domain/order"
)

type Dispatcher struct {
	publisher *Publisher
}

func NewDispatcher(
	publisher *Publisher,
) *Dispatcher {
	return &Dispatcher{
		publisher: publisher,
	}
}

func (d *Dispatcher) DispatchOrderAccepted(
	o *order.Order,
) {

	report := ExecutionReport{
		OrderID:           o.ID,
		Symbol:            o.Symbol,
		Status:            string(o.Status),
		Price:             o.Price,
		Quantity:          o.Quantity,
		FilledQuantity:    o.Filled,
		RemainingQuantity: o.Remaining,
	}

	d.publisher.PublishOrderAccepted(
		fmt.Sprint(o.UserID),
		report,
	)
}

func (d *Dispatcher) DispatchOrderFilled(
	o *order.Order,
) {

	report := ExecutionReport{
		OrderID:           o.ID,
		Symbol:            o.Symbol,
		Status:            string(o.Status),
		Price:             o.Price,
		Quantity:          o.Quantity,
		FilledQuantity:    o.Filled,
		RemainingQuantity: o.Remaining,
	}

	d.publisher.PublishOrderFilled(
		fmt.Sprint(o.UserID),
		report,
	)
}

func (d *Dispatcher) DispatchOrderCancelled(
	o *order.Order,
) {

	report := ExecutionReport{
		OrderID:           o.ID,
		Symbol:            o.Symbol,
		Status:            string(o.Status),
		Price:             o.Price,
		Quantity:          o.Quantity,
		FilledQuantity:    o.Filled,
		RemainingQuantity: o.Remaining,
	}

	d.publisher.PublishOrderCancelled(
		fmt.Sprint(o.UserID),
		report,
	)
}

func (d *Dispatcher) DispatchOrderRejected(
	o *order.Order,
) {

	report := ExecutionReport{
		OrderID:           o.ID,
		Symbol:            o.Symbol,
		Status:            string(o.Status),
		Price:             o.Price,
		Quantity:          o.Quantity,
		FilledQuantity:    o.Filled,
		RemainingQuantity: o.Remaining,
	}

	d.publisher.PublishOrderRejected(
		fmt.Sprint(o.UserID),
		report,
	)
}


func (d *Dispatcher) DispatchOrderModified(
	o *order.Order,
) {

	report := ExecutionReport{
		OrderID:           o.ID,
		Symbol:            o.Symbol,
		Status:            string(o.Status),
		Price:             o.Price,
		Quantity:          o.Quantity,
		FilledQuantity:    o.Filled,
		RemainingQuantity: o.Remaining,
	}

	d.publisher.PublishOrderModified(
		fmt.Sprint(o.UserID),
		report,
	)
}