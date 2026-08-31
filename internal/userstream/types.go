package userstream

type EventType string

const (

	// -------------------------
	// Order Events
	// -------------------------

	EventOrderAccepted        EventType = "order.accepted"
	EventOrderRejected        EventType = "order.rejected"
	EventOrderCancelled       EventType = "order.cancelled"
	EventOrderModified        EventType = "order.modified"
	EventOrderPartiallyFilled EventType = "order.partially_filled"
	EventOrderFilled          EventType = "order.filled"

	// -------------------------
	// Trade Events
	// -------------------------

	EventTradeExecuted EventType = "trade.executed"

	// -------------------------
	// Account Events
	// -------------------------

	EventBalanceUpdated  EventType = "balance.updated"
	EventPositionUpdated EventType = "position.updated"

	// -------------------------
	// Connection Events
	// -------------------------

	EventPing EventType = "ping"
	EventPong EventType = "pong"
)
