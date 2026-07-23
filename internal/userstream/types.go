package userstream

type EventType string

const (
	EventOrderAccepted EventType = "order_accepted"
	EventOrderRejected EventType = "order_rejected"
	EventOrderCancelled EventType = "order_cancelled"
	EventOrderModified EventType = "order_modified"
	EventOrderPartiallyFilled EventType = "order_partially_filled"
	EventOrderFilled EventType = "order_filled"

	EventBalanceUpdated EventType = "balance_updated"
	EventPositionUpdated EventType = "position_updated"

	EventPing EventType = "ping"
	EventPong EventType = "pong"
    
    MessageTypeOrderModified EventType = "order.modified"
)