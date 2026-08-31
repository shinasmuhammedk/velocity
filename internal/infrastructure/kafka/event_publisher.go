package kafka

import (
	"context"
	"time"

	"go.uber.org/zap"

	"velocity/internal/engine/events"
	"velocity/internal/infrastructure/metrics"
)

const (
	// publishQueueSize bounds how many events can be buffered waiting
	// to reach Kafka. Sized generously above normal burst load - once
	// full, new events are dropped (and logged) rather than blocking
	// the caller, since the caller here is the matching engine's
	// single-writer goroutine and must never stall on Kafka.
	publishQueueSize = 4096

	// publishTimeout bounds a single write to Kafka. Without this, a
	// broker outage or network partition could hang the publish
	// goroutine indefinitely; with it, a stuck write is abandoned and
	// logged instead.
	publishTimeout = 5 * time.Second
)

// EventPublisher bridges the in-process event dispatcher to Kafka.
//
// Handle() is called synchronously from the matching engine's
// single-writer goroutine (see internal/engine/engine.go, ADR-002),
// so it must never block on network I/O. It only ever does a
// non-blocking channel send; the actual Kafka write happens on a
// separate goroutine, so a slow or unreachable broker can never
// stall order matching.
type EventPublisher struct {
	producer *Producer
	topic    string
	logger   *zap.Logger

	queue chan queuedEvent
	done  chan struct{}
}

type queuedEvent struct {
	key      string
	envelope EventEnvelope
}

func NewEventPublisher(
	producer *Producer,
	topic string,
	logger *zap.Logger,
) *EventPublisher {

	p := &EventPublisher{
		producer: producer,
		topic:    topic,
		logger:   logger,
		queue:    make(chan queuedEvent, publishQueueSize),
		done:     make(chan struct{}),
	}

	go p.run()

	return p
}

// Handle implements events.Subscriber. It must stay non-blocking -
// see the EventPublisher doc comment for why.
func (p *EventPublisher) Handle(event events.Event) {

	key := eventKey(event)

	envelope := EventEnvelope{
		Type:      string(event.Type()),
		Timestamp: event.Timestamp(),
		Symbol:    key,
		Payload:   event,
	}

	select {
	case p.queue <- queuedEvent{key: key, envelope: envelope}:

	default:
		// Queue is full - the publish side can't keep up, or Kafka is
		// down. Drop rather than block the matching engine; this
		// event is lost from the Kafka stream, but matching itself
		// is never affected.
		metrics.KafkaEventQueueDropped.Inc()

		p.logger.Warn(
			"kafka event queue full, dropping event",
			zap.String("type", envelope.Type),
			zap.String("key", key),
		)
	}
}

// run drains the queue and performs the actual Kafka writes. It runs
// on its own goroutine, entirely decoupled from whatever called
// Handle().
func (p *EventPublisher) run() {
	for {
		select {
		case qe := <-p.queue:
			p.publish(qe)

		case <-p.done:
			return
		}
	}
}

func (p *EventPublisher) publish(qe queuedEvent) {

	ctx, cancel := context.WithTimeout(
		context.Background(),
		publishTimeout,
	)
	defer cancel()

	if err := p.producer.Publish(
		ctx,
		qe.key,
		qe.envelope,
	); err != nil {
		p.logger.Error(
			"kafka event publish failed",
			zap.String("type", qe.envelope.Type),
			zap.String("key", qe.key),
			zap.Error(err),
		)
		return
	}

	p.logger.Debug(
		"kafka event published",
		zap.String("type", qe.envelope.Type),
		zap.String("key", qe.key),
	)
}

// Close stops the publish goroutine. Events still sitting in the
// queue when Close is called are not flushed - callers that need a
// best-effort drain should give the app a moment to work through the
// queue before calling Close during shutdown.
func (p *EventPublisher) Close() {
	close(p.done)
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
