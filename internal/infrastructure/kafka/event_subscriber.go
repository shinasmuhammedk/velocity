package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"velocity/internal/engine/events"

	"github.com/segmentio/kafka-go"
)

type EventSubscriber struct {
	dispatcher *events.Dispatcher
}

func NewEventSubscriber(
	dispatcher *events.Dispatcher,
) *EventSubscriber {
	return &EventSubscriber{
		dispatcher: dispatcher,
	}
}

func (s *EventSubscriber) HandleMessage(
	ctx context.Context,
	message kafka.Message,
) error {
	var envelope EventEnvelope

	if err := json.Unmarshal(message.Value, &envelope); err != nil {
		return fmt.Errorf("decode event envelope: %w", err)
	}

	event, err := decodeEvent(envelope)
	if err != nil {
		return err
	}

	fmt.Println(
		"KAFKA EVENT RECEIVED:",
		event.Type(),
		"key:",
		string(message.Key),
	)

	s.dispatcher.Publish(event)

	return nil
}

func decodeEvent(envelope EventEnvelope) (events.Event, error) {
	payload, err := json.Marshal(envelope.Payload)
	if err != nil {
		return nil, fmt.Errorf(
			"encode event payload: %w",
			err,
		)
	}

	baseEvent := events.BaseEvent{
		Time: envelope.Timestamp,
	}

	switch events.EventType(envelope.Type) {

	case events.TradeExecutedEventType:
		var event events.TradeExecutedEvent

		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf(
				"decode trade.executed event: %w",
				err,
			)
		}

		event.BaseEvent = baseEvent

		return event, nil

	case events.OrderAcceptedEventType:
		var event events.OrderAcceptedEvent

		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf(
				"decode order.accepted event: %w",
				err,
			)
		}

		event.BaseEvent = baseEvent

		return event, nil

	case events.OrderRejectedEventType:
		var event events.OrderRejectedEvent

		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf(
				"decode order.rejected event: %w",
				err,
			)
		}

		event.BaseEvent = baseEvent

		return event, nil

	case events.OrderCancelledEventType:
		var event events.OrderCancelledEvent

		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf(
				"decode order.cancelled event: %w",
				err,
			)
		}

		event.BaseEvent = baseEvent

		return event, nil

	case events.OrderModifiedEventType:
		var event events.OrderModifiedEvent

		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf(
				"decode order.modified event: %w",
				err,
			)
		}

		event.BaseEvent = baseEvent

		return event, nil

	case events.OrderTriggeredEventType:
		var event events.OrderTriggeredEvent

		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf(
				"decode order.triggered event: %w",
				err,
			)
		}

		event.BaseEvent = baseEvent

		return event, nil

	case events.DepthUpdatedEventType:
		var event events.DepthUpdatedEvent

		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf(
				"decode depth.updated event: %w",
				err,
			)
		}

		event.BaseEvent = baseEvent

		return event, nil

	case events.TickerUpdatedEventType:
		var event events.TickerUpdatedEvent

		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf(
				"decode ticker.updated event: %w",
				err,
			)
		}

		event.BaseEvent = baseEvent

		return event, nil

	default:
		return nil, fmt.Errorf(
			"unsupported event type: %s",
			envelope.Type,
		)
	}
}
