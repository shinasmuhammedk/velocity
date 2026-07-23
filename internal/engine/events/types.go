package events

type EventType string

const (
	OrderAcceptedEventType       EventType = "order.accepted"
	OrderRejectedEventType       EventType = "order.rejected"
	OrderCancelledEventType      EventType = "order.cancelled"
	OrderModifiedEventType       EventType = "order.modified"

	OrderPartiallyFilledEventType EventType = "order.partially_filled"
	OrderFilledEventType          EventType = "order.filled"

	TradeExecutedEventType EventType = "trade.executed"

	DepthUpdatedEventType  EventType = "depth.updated"
	TickerUpdatedEventType EventType = "ticker.updated"
)