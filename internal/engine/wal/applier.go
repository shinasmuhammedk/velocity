package wal

import (
	"velocity/internal/engine/orderbook"
	"velocity/internal/engine/stopbook"
	"velocity/pkg/constants"
)

type Applier struct {
	book     *orderbook.OrderBook
	stopBook *stopbook.StopBook
}

func NewApplier(
	book *orderbook.OrderBook,
	stopBook *stopbook.StopBook,
) *Applier {
	return &Applier{
		book:     book,
		stopBook: stopBook,
	}
}

func (a *Applier) Apply(event *Event) error {
	switch event.Type {

	case EventSubmit:
		return a.applySubmit(event)

	case EventCancel:
		return a.applyCancel(event)

	case EventModify:
		return a.applyModify(event)

	default:
		return nil
	}
}

func (a *Applier) applySubmit(event *Event) error {
	if event.Order == nil {
		return nil
	}

	switch event.Order.Type {

	case constants.StopMarketOrder,
		constants.StopLimitOrder:

		a.stopBook.Add(event.Order)

	default:

		a.book.AddOrder(event.Order)
	}

	return nil
}

func (a *Applier) applyCancel(event *Event) error {

	// First check whether it is a stop order.
	if err := a.stopBook.CancelOrder(event.OrderID); err == nil {
		return nil
	}

	// Otherwise it is a normal order.
	_, err := a.book.CancelOrder(event.OrderID)
	return err
}

func (a *Applier) applyModify(event *Event) error {

	return a.book.ModifyOrder(
		event.OrderID,
		event.NewPrice,
		event.NewQuantity,
	)
}
