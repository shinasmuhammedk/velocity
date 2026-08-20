package kafka

import (
	"testing"
	"time"

	"velocity/internal/engine/events"
)

func TestDecodeEvent_DepthUpdated(t *testing.T) {
	timestamp := time.Now().UTC()

	envelope := EventEnvelope{
		Type:      string(events.DepthUpdatedEventType),
		Timestamp: timestamp,
		Symbol:    "BTCUSDT",
		Payload: map[string]interface{}{
			"Symbol": "BTCUSDT",
		},
	}

	event, err := decodeEvent(envelope)
	if err != nil {
		t.Fatalf("decodeEvent() error = %v", err)
	}

	depthEvent, ok := event.(events.DepthUpdatedEvent)
	if !ok {
		t.Fatalf(
			"expected events.DepthUpdatedEvent, got %T",
			event,
		)
	}

	if depthEvent.Symbol != "BTCUSDT" {
		t.Fatalf(
			"expected symbol BTCUSDT, got %s",
			depthEvent.Symbol,
		)
	}

	if !depthEvent.Timestamp().Equal(timestamp) {
		t.Fatalf(
			"expected timestamp %v, got %v",
			timestamp,
			depthEvent.Timestamp(),
		)
	}
}

func TestDecodeEvent_TickerUpdated(t *testing.T) {
	timestamp := time.Now().UTC()

	envelope := EventEnvelope{
		Type:      string(events.TickerUpdatedEventType),
		Timestamp: timestamp,
		Symbol:    "BTCUSDT",
		Payload: map[string]interface{}{
			"Symbol":    "BTCUSDT",
			"LastPrice": int64(300000),
			"Volume":    int64(5),
		},
	}

	event, err := decodeEvent(envelope)
	if err != nil {
		t.Fatalf("decodeEvent() error = %v", err)
	}

	tickerEvent, ok := event.(events.TickerUpdatedEvent)
	if !ok {
		t.Fatalf(
			"expected events.TickerUpdatedEvent, got %T",
			event,
		)
	}

	if tickerEvent.Symbol != "BTCUSDT" {
		t.Fatalf(
			"expected symbol BTCUSDT, got %s",
			tickerEvent.Symbol,
		)
	}

	if tickerEvent.LastPrice != 300000 {
		t.Fatalf(
			"expected LastPrice 300000, got %d",
			tickerEvent.LastPrice,
		)
	}

	if tickerEvent.Volume != 5 {
		t.Fatalf(
			"expected Volume 5, got %d",
			tickerEvent.Volume,
		)
	}

	if !tickerEvent.Timestamp().Equal(timestamp) {
		t.Fatalf(
			"expected timestamp %v, got %v",
			timestamp,
			tickerEvent.Timestamp(),
		)
	}
}
