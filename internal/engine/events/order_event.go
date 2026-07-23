package events


type OrderAcceptedEvent struct {
	BaseEvent

	OrderID string
	UserID  string
	Symbol  string

	Price    int64
	Quantity int64
}

func (e OrderAcceptedEvent) Type() EventType {
	return OrderAcceptedEventType
}

// ----------------------------------------------------

type OrderRejectedEvent struct {
	BaseEvent

	OrderID string
	UserID  string
	Symbol  string

	Reason string
}

func (e OrderRejectedEvent) Type() EventType {
	return OrderRejectedEventType
}

// ----------------------------------------------------

type OrderCancelledEvent struct {
	BaseEvent

	OrderID string
	Symbol  string
	UserID  string
}

func (e OrderCancelledEvent) Type() EventType {
	return OrderCancelledEventType

}

// ----------------------------------------------------

type OrderModifiedEvent struct {
	BaseEvent

	OrderID string
	Symbol  string

	NewPrice    int64
	NewQuantity int64
}

func (e OrderModifiedEvent) Type() EventType {
	return OrderModifiedEventType

}

// ----------------------------------------------------

type OrderTriggeredEvent struct {
	BaseEvent

	OrderID string
	Symbol  string
}

func (e OrderTriggeredEvent) Type() EventType {
	return OrderTriggeredEventType
}