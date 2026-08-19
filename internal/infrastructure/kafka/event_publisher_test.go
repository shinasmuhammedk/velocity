
package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go.uber.org/zap"

	"velocity/internal/engine/events"

	"github.com/segmentio/kafka-go"
)

func TestEventPublisher(t *testing.T) {
	producer := NewProducer(
		[]string{"localhost:9092"},
		"velocity-events-test",
	)

	defer producer.Close()

	publisher := NewEventPublisher(
		producer,
		"velocity-events-test",
        zap.NewNop(),
	)

	event := events.TradeExecutedEvent{
		BaseEvent: events.NewBaseEvent(),

		TradeID: 1,

		BuyOrderID:  1001,
		SellOrderID: 1002,

		BuyerID:  10,
		SellerID: 20,

		Symbol:   "BTCUSDT",
		Price:    300000,
		Quantity: 1,
	}

	publisher.Handle(event)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "velocity-events-test",

		GroupID: "velocity-event-publisher-test-" +
			time.Now().Format("20060102150405.000000000"),

		StartOffset: kafka.FirstOffset,
	})

	defer reader.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	var envelope EventEnvelope

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			t.Fatalf(
				"failed to read Kafka message: %v",
				err,
			)
		}

		t.Logf(
			"received message: key=%s value=%s",
			string(msg.Key),
			string(msg.Value),
		)

		if string(msg.Key) != "BTCUSDT" {
			continue
		}

		var candidate EventEnvelope

		if err := json.Unmarshal(msg.Value, &candidate); err != nil {
			continue
		}

		if candidate.Type != string(events.TradeExecutedEventType) {
			continue
		}

		if candidate.Symbol != "BTCUSDT" {
			continue
		}

		envelope = candidate
		break
	}

	if envelope.Type != string(events.TradeExecutedEventType) {
		t.Fatalf(
			"expected event type %s, got %s",
			events.TradeExecutedEventType,
			envelope.Type,
		)
	}

	if envelope.Symbol != "BTCUSDT" {
		t.Fatalf(
			"expected symbol BTCUSDT, got %s",
			envelope.Symbol,
		)
	}

	if envelope.Timestamp.IsZero() {
		t.Fatal("expected timestamp to be set")
	}

	payload, ok := envelope.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf(
			"expected payload to be an object, got %T",
			envelope.Payload,
		)
	}

	if int64(payload["TradeID"].(float64)) != 1 {
		t.Fatalf(
			"expected TradeID 1, got %v",
			payload["TradeID"],
		)
	}

	if int64(payload["BuyOrderID"].(float64)) != 1001 {
		t.Fatalf(
			"expected BuyOrderID 1001, got %v",
			payload["BuyOrderID"],
		)
	}

	if int64(payload["SellOrderID"].(float64)) != 1002 {
		t.Fatalf(
			"expected SellOrderID 1002, got %v",
			payload["SellOrderID"],
		)
	}

	if payload["Symbol"] != "BTCUSDT" {
		t.Fatalf(
			"expected payload Symbol BTCUSDT, got %v",
			payload["Symbol"],
		)
	}

	if int64(payload["Price"].(float64)) != 300000 {
		t.Fatalf(
			"expected Price 300000, got %v",
			payload["Price"],
		)
	}

	if int64(payload["Quantity"].(float64)) != 1 {
		t.Fatalf(
			"expected Quantity 1, got %v",
			payload["Quantity"],
		)
	}
}
