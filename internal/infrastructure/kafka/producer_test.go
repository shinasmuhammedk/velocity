package kafka

import (
	"context"
	"testing"
	"time"
)

func TestProducerPublish(t *testing.T) {
	producer := NewProducer(
		[]string{"localhost:9092"},
		"velocity-events-test",
	)

	defer producer.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	err := producer.Publish(
		ctx,
		"BTCUSDT",
		map[string]any{
			"type":   "test",
			"symbol": "BTCUSDT",
			"message": "hello kafka",
		},
	)

	if err != nil {
		t.Fatalf("failed to publish message: %v", err)
	}
}