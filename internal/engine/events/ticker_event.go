package events

type TickerUpdatedEvent struct {
	BaseEvent

	Symbol string

	LastPrice int64

	Volume int64
}

func (e TickerUpdatedEvent) Type() EventType {
	return TickerUpdatedEventType
}