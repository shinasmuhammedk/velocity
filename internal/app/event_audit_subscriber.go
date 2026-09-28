package app

import (
	"velocity/internal/engine/events"
	"velocity/pkg/logger"

	"go.uber.org/zap"
)

// EventAuditSubscriber listens for events delivered via Kafka and logs them.
// It is NOT responsible for settlement or any state mutation — settlement
// already happens synchronously in-process (see
// internal/persistence/worker.TradeConsumer, wired to the engine's
// tradeQueue channel). This subscriber exists to prove the Kafka pipeline
// works end-to-end for every event type, and as a seam for future consumers
// (audit logging, notifications, analytics) that don't touch the database
// of record.
type EventAuditSubscriber struct {
	log *zap.Logger
}

func NewEventAuditSubscriber() *EventAuditSubscriber {
	return &EventAuditSubscriber{
		log: logger.Logger(),
	}
}

// Handle implements events.Subscriber.
func (s *EventAuditSubscriber) Handle(event events.Event) {
	switch e := event.(type) {

	case events.TradeExecutedEvent:
		s.log.Info(
			"worker audit: trade.executed",
			zap.Int64("trade_id", e.TradeID),
			zap.String("symbol", e.Symbol),
			zap.Int64("price", e.Price),
			zap.Int64("quantity", e.Quantity),
		)

	case events.OrderAcceptedEvent:
		s.log.Info(
			"worker audit: order.accepted",
			zap.String("order_id", e.OrderID),
			zap.String("user_id", e.UserID),
			zap.String("symbol", e.Symbol),
			zap.Int64("price", e.Price),
			zap.Int64("quantity", e.Quantity),
		)

	case events.OrderRejectedEvent:
		s.log.Info(
			"worker audit: order.rejected",
			zap.String("order_id", e.OrderID),
			zap.String("user_id", e.UserID),
			zap.String("symbol", e.Symbol),
			zap.String("reason", e.Reason),
		)

	case events.OrderCancelledEvent:
		s.log.Info(
			"worker audit: order.cancelled",
			zap.Int64("order_id", e.OrderID),
			zap.Int64("user_id", e.UserID),
			zap.String("symbol", e.Symbol),
		)

	case events.OrderModifiedEvent:
		s.log.Info(
			"worker audit: order.modified",
			zap.Int64("order_id", e.OrderID),
			zap.String("symbol", e.Symbol),
			zap.Int64("new_price", e.NewPrice),
			zap.Int64("new_quantity", e.NewQuantity),
		)

	case events.OrderTriggeredEvent:
		s.log.Info(
			"worker audit: order.triggered",
			zap.String("order_id", e.OrderID),
			zap.String("symbol", e.Symbol),
		)

	default:
		s.log.Warn(
			"worker audit: unrecognized event type received",
			zap.String("event_type", string(event.Type())),
		)
	}
}
