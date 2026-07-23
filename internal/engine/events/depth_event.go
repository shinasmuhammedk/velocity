package events

type DepthUpdatedEvent struct {
	BaseEvent

	Symbol string
}

func (e DepthUpdatedEvent) Type() EventType {
	return DepthUpdatedEventType
}