package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

func TestProducerPublish(t *testing.T) {
	brokers := []string{"localhost:9092"}
	topic := "velocity-producer-test-" + uuid.NewString()[:8]

	// This broker has topic auto-creation disabled, so the topic must
	// be provisioned explicitly before Publish can succeed - see
	// TestEnsureTopics in this package for the same helper.
	require.NoError(t, EnsureTopics(brokers, TopicConfig{
		Name:              topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}))

	defer func() {
		conn, err := kafka.Dial("tcp", brokers[0])
		if err != nil {
			t.Logf("cleanup: failed to dial kafka: %v", err)
			return
		}
		defer conn.Close()
		if err := conn.DeleteTopics(topic); err != nil {
			t.Logf("cleanup: failed to delete test topic %s: %v", topic, err)
		}
	}()

	producer := NewProducer(
		brokers,
		topic,
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
			"type":    "test",
			"symbol":  "BTCUSDT",
			"message": "hello kafka",
		},
	)

	if err != nil {
		t.Fatalf("failed to publish message: %v", err)
	}
}