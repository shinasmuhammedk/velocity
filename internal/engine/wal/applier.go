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

	// Idempotency guard: a SUBMIT event replayed a second time for an
	// order ID already resting in the book must be a no-op, not a
	// second physical entry. orderbook.AddOrder / stopBook.Add have no
	// such guard themselves (they're the hot path and shouldn't pay for
	// a lookup on every live order), so this is the one place a replay
	// - from any cause, not just the snapshot-recovery race this was
	// written for - can't silently duplicate a resting order.
	switch event.Order.Type {

	case constants.StopMarketOrder,
		constants.StopLimitOrder:

		if a.stopBook.Contains(event.Order.ID) {
			return nil
		}

	default:

		if a.book.GetOrder(event.Order.ID) != nil {
			return nil
		}
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