package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

func TestConsumer(t *testing.T) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	brokers := []string{"localhost:9092"}
	topic := "velocity-consumer-test-" + uuid.NewString()[:8]

	// This broker has topic auto-creation disabled (see
	// deployments/compose/docker-compose.yml and the CI workflow), so
	// the topic must be provisioned explicitly - same helper the chaos
	// and DLQ tests in this package already use. A test-unique topic
	// (rather than a fixed shared name) also means this test can never
	// see a stale message left behind by a previous run or by
	// TestEventPublisher/TestProducerPublish racing against it.
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

	received := make(chan kafka.Message, 1)

	consumer := NewConsumer(
		brokers,
		topic,
		"velocity-test-group-"+uuid.NewString()[:8],
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
		brokers,
		topic,
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