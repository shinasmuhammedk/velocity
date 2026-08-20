package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestConsumer(t *testing.T) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	received := make(chan kafka.Message, 1)

	consumer := NewConsumer(
		[]string{"localhost:9092"},
		"velocity-events-test",
		"velocity-test-group",
		func(ctx context.Context, message kafka.Message) error {
			received <- message
			return nil
		},
        nil,
	)

	defer consumer.Close()

	go func() {
		if err := consumer.Start(ctx); err != nil {
			t.Logf("consumer stopped: %v", err)
		}
	}()

	// Give the consumer time to connect.
	time.Sleep(1 * time.Second)

	producer := NewProducer(
		[]string{"localhost:9092"},
		"velocity-events-test",
	)

	defer producer.Close()

	err := producer.Publish(
		ctx,
		"BTCUSDT",
		map[string]any{
			"type":    "consumer.test",
			"symbol":  "BTCUSDT",
			"message": "hello consumer",
		},
	)

	if err != nil {
		t.Fatalf("failed to publish message: %v", err)
	}

	select {
	case message := <-received:
		t.Logf(
			"received message: key=%s value=%s",
			string(message.Key),
			string(message.Value),
		)

	case <-ctx.Done():
		t.Fatal("timed out waiting for Kafka message")
	}
}