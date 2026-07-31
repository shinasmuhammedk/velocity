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
	d.publisher.PublishOrderAccepted(
		fmt.Sprint(o.UserID),
		newExecutionReport(o),
	)
}

func (d *Dispatcher) DispatchOrderFilled(
	o *order.Order,
) {

	d.publisher.PublishOrderFilled(
		fmt.Sprint(o.UserID),
		newExecutionReport(o),
	)
}

func (d *Dispatcher) DispatchOrderCancelled(
	o *order.Order,
) {

	d.publisher.PublishOrderCancelled(
		fmt.Sprint(o.UserID),
		newExecutionReport(o),
	)
}

func (d *Dispatcher) DispatchOrderRejected(
	o *order.Order,
) {

	d.publisher.PublishOrderRejected(
		fmt.Sprint(o.UserID),
		newExecutionReport(o),
	)
}

func (d *Dispatcher) DispatchOrderModified(
	o *order.Order,
) {

	d.publisher.PublishOrderModified(
		fmt.Sprint(o.UserID),
		newExecutionReport(o),
	)
}

func (d *Dispatcher) DispatchTradeExecuted(
	trade TradeExecution,
) {
	d.publisher.PublishTradeExecuted(
		trade.BuyerID,
		trade,
	)

	d.publisher.PublishTradeExecuted(
		trade.SellerID,
		trade,
	)
}

func (d *Dispatcher) DispatchBalanceUpdated(
	userID string,
	update BalanceUpdate,
) {
	d.publisher.PublishBalanceUpdate(
		userID,
		update,
	)
}

func (d *Dispatcher) DispatchPositionUpdated(
	userID string,
	update PositionUpdate,
) {
	d.publisher.PublishPositionUpdate(
		userID,
		update,
	)
}

func (d *Dispatcher) DispatchOrderPartiallyFilled(
	o *order.Order,
) {

	d.publisher.PublishOrderPartiallyFilled(
		fmt.Sprint(o.UserID),
		newExecutionReport(o),
	)
}

func newExecutionReport(
	o *order.Order,
) ExecutionReport {

	return ExecutionReport{
		OrderID:           o.ID,
		Symbol:            o.Symbol,
		Status:            string(o.Status),
		Price:             o.Price,
		Quantity:          o.Quantity,
		FilledQuantity:    o.Filled,
		RemainingQuantity: o.Remaining,
	}
}
