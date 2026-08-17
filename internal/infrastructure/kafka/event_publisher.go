package kafka

import (
	"context"
	"fmt"

	"velocity/internal/engine/events"
)

type EventPublisher struct {
	producer *Producer
	topic    string
}

func NewEventPublisher(
	producer *Producer,
	topic string,
) *EventPublisher {
	return &EventPublisher{
		producer: producer,
		topic:    topic,
	}
}

func (p *EventPublisher) Handle(event events.Event) {
	key := eventKey(event)

	envelope := EventEnvelope{
		Type:      string(event.Type()),
		Timestamp: event.Timestamp(),
		Symbol:    key,
		Payload:   event,
	}

	if err := p.producer.Publish(
		context.Background(),
		key,
		envelope,
	); err != nil {
		fmt.Println(
			"KAFKA EVENT PUBLISH ERROR:",
			event.Type(),
			err,
		)
		return
	}

	fmt.Println(
		"KAFKA EVENT PUBLISHED:",
		event.Type(),
		"key:",
		key,
	)
}

func eventKey(event events.Event) string {
	switch e := event.(type) {
	case events.TradeExecutedEvent:
		return e.Symbol

	case events.OrderAcceptedEvent:
		return e.Symbol

	case events.OrderRejectedEvent:
		return e.Symbol

	case events.OrderCancelledEvent:
		return e.Symbol

	case events.OrderModifiedEvent:
		return e.Symbol

	case events.OrderTriggeredEvent:
		return e.Symbol

	case events.DepthUpdatedEvent:
		return e.Symbol

	case events.TickerUpdatedEvent:
		return e.Symbol

	default:
		return ""
	}
}
