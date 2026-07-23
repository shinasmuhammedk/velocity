package events

type OrderAcceptedEvent struct {
	BaseEvent

	OrderID string
	UserID  string
	Symbol  string
}

func (e OrderAcceptedEvent) Type() EventType {
	return OrderAcceptedEventType
}

type OrderRejectedEvent struct {
	BaseEvent

	OrderID string
	UserID  string
	Symbol  string
	Reason  string
}

func (e OrderRejectedEvent) Type() EventType {
	return OrderRejectedEventType
}

type OrderCancelledEvent struct {
	BaseEvent

	OrderID string
	UserID  string
	Symbol  string
}

func (e OrderCancelledEvent) Type() EventType {
	return OrderCancelledEventType
}

type OrderModifiedEvent struct {
	BaseEvent

	OrderID string
	UserID  string
	Symbol  string
}

func (e OrderModifiedEvent) Type() EventType {
	return OrderModifiedEventType
}

type OrderPartiallyFilledEvent struct {
	BaseEvent

	OrderID        string
	UserID         string
	Symbol         string
	FilledQuantity int64
	RemainingQty   int64
}

func (e OrderPartiallyFilledEvent) Type() EventType {
	return OrderPartiallyFilledEventType
}

type OrderFilledEvent struct {
	BaseEvent

	OrderID string
	UserID  string
	Symbol  string
}

func (e OrderFilledEvent) Type() EventType {
	return OrderFilledEventType
}