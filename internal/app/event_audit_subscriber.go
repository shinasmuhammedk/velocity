package app

import (
	"fmt"

	"velocity/internal/engine/events"
)

// EventAuditSubscriber listens for events delivered via Kafka and logs them.
// It is NOT responsible for settlement or any state mutation — settlement
// already happens synchronously in-process (see
// internal/persistence/worker.TradeConsumer, wired to the engine's
// tradeQueue channel). This subscriber exists to prove the Kafka pipeline
// works end-to-end for every event type, and as a seam for future consumers
// (audit logging, notifications, analytics) that don't touch the database
// of record.
type EventAuditSubscriber struct{}

func NewEventAuditSubscriber() *EventAuditSubscriber {
	return &EventAuditSubscriber{}
}

// Handle implements events.Subscriber.
func (s *EventAuditSubscriber) Handle(event events.Event) {
	switch e := event.(type) {

	case events.TradeExecutedEvent:
		fmt.Println(
			"WORKER AUDIT: trade.executed",
			"tradeID:", e.TradeID,
			"symbol:", e.Symbol,
			"price:", e.Price,
			"quantity:", e.Quantity,
		)

	case events.OrderAcceptedEvent:
		fmt.Println(
			"WORKER AUDIT: order.accepted",
			"orderID:", e.OrderID,
			"userID:", e.UserID,
			"symbol:", e.Symbol,
			"price:", e.Price,
			"quantity:", e.Quantity,
		)

	case events.OrderRejectedEvent:
		fmt.Println(
			"WORKER AUDIT: order.rejected",
			"orderID:", e.OrderID,
			"userID:", e.UserID,
			"symbol:", e.Symbol,
			"reason:", e.Reason,
		)

	case events.OrderCancelledEvent:
		fmt.Println(
			"WORKER AUDIT: order.cancelled",
			"orderID:", e.OrderID,
			"userID:", e.UserID,
			"symbol:", e.Symbol,
		)

	case events.OrderModifiedEvent:
		fmt.Println(
			"WORKER AUDIT: order.modified",
			"orderID:", e.OrderID,
			"symbol:", e.Symbol,
			"newPrice:", e.NewPrice,
			"newQuantity:", e.NewQuantity,
		)

	case events.OrderTriggeredEvent:
		fmt.Println(
			"WORKER AUDIT: order.triggered",
			"orderID:", e.OrderID,
			"symbol:", e.Symbol,
		)

	default:
		fmt.Println("WORKER AUDIT: unrecognized event type received:", event.Type())
	}
}