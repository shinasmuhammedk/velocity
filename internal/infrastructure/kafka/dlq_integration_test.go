package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

var errTestMessage = errors.New(
	"intentional DLQ test failure",
)

// waitForKafkaTopic waits until Kafka exposes the topic metadata.
//
// Creating a topic and being able to immediately produce to it are
// not necessarily the same moment. Kafka may need some time to
// propagate the topic metadata. This helper removes that race from
// the integration test.
func waitForKafkaTopic(
	ctx context.Context,
	brokers []string,
	topic string,
) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		conn, err := kafka.Dial("tcp", brokers[0])
		if err == nil {
			partitions, partitionErr := conn.ReadPartitions(topic)
			conn.Close()

			if partitionErr == nil && len(partitions) > 0 {
				leaderConn, leaderErr := kafka.DialLeader(
					ctx,
					"tcp",
					brokers[0],
					topic,
					0,
				)

				if leaderErr == nil {
					leaderConn.Close()
					return nil
				}

				if leaderConn != nil {
					leaderConn.Close()
				}
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"timeout waiting for Kafka topic %s leader: %w",
				topic,
				ctx.Err(),
			)

		case <-ticker.C:
		}
	}
}

// waitForKafkaConnection waits until the Kafka broker itself is reachable.
func waitForKafkaConnection(
	ctx context.Context,
	brokers []string,
) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		conn, err := kafka.Dial("tcp", brokers[0])
		if err == nil {
			conn.Close()
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"timeout waiting for Kafka broker: %w",
				ctx.Err(),
			)

		case <-ticker.C:
		}
	}
}

func TestKafkaDLQ_EndToEnd(t *testing.T) {
	brokers := []string{"localhost:9092"}

	// ------------------------------------------------------------
	// Test context
	// ------------------------------------------------------------

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	// ------------------------------------------------------------
	// Kafka broker readiness
	// ------------------------------------------------------------

	if err := waitForKafkaConnection(ctx, brokers); err != nil {
		t.Fatalf(
			"Kafka broker is not ready: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// Unique test topics
	// ------------------------------------------------------------

	testID := time.Now().Format(
		"20060102150405.000000000",
	)

	sourceTopic := "velocity-events-dlq-test-" + testID
	dlqTopic := sourceTopic + "-dlq"

	// ------------------------------------------------------------
	// Topic provisioning
	// ------------------------------------------------------------

	testTopics := []string{
		sourceTopic,
		dlqTopic,
	}

	for _, topic := range testTopics {
		err := EnsureTopics(
			brokers,
			TopicConfig{
				Name:              topic,
				NumPartitions:     1,
				ReplicationFactor: 1,
			},
		)
		if err != nil {
			t.Fatalf(
				"failed to provision Kafka topic %s: %v",
				topic,
				err,
			)
		}

		// Ensure Kafka metadata has propagated before continuing.
		if err := waitForKafkaTopic(
			ctx,
			brokers,
			topic,
		); err != nil {
			t.Fatalf(
				"Kafka topic %s did not become ready: %v",
				topic,
				err,
			)
		}
	}

	// ------------------------------------------------------------
	// Cleanup
	// ------------------------------------------------------------

	t.Cleanup(func() {
		conn, err := kafka.Dial(
			"tcp",
			brokers[0],
		)
		if err != nil {
			t.Logf(
				"failed to connect for topic cleanup: %v",
				err,
			)
			return
		}

		defer conn.Close()

		if err := conn.DeleteTopics(
			sourceTopic,
			dlqTopic,
		); err != nil {
			t.Logf(
				"failed to delete test topics: %v",
				err,
			)
		}
	})

	// ------------------------------------------------------------
	// DLQ producer
	// ------------------------------------------------------------

	dlqProducer := NewProducer(
		brokers,
		dlqTopic,
	)

	t.Cleanup(func() {
		if err := dlqProducer.Close(); err != nil {
			t.Logf(
				"failed to close DLQ producer: %v",
				err,
			)
		}
	})

	dlqPublisher := NewKafkaDLQPublisher(
		dlqProducer,
	)

	// ------------------------------------------------------------
	// Consumer
	// ------------------------------------------------------------

	consumerGroup := "velocity-dlq-test-" + testID

	consumer := NewConsumer(
		brokers,
		sourceTopic,
		consumerGroup,
		func(
			ctx context.Context,
			message kafka.Message,
		) error {
			// Deliberately fail every message.
			//
			// The Consumer must send this message to the DLQ
			// instead of silently dropping it.
			return errTestMessage
		},
		dlqPublisher,
	)

	t.Cleanup(func() {
		if err := consumer.Close(); err != nil {
			t.Logf(
				"failed to close consumer: %v",
				err,
			)
		}
	})

	consumerDone := make(chan error, 1)

	go func() {
		consumerDone <- consumer.Start(ctx)
	}()

	// ------------------------------------------------------------
	// Give the consumer a moment to establish its connection.
	//
	// This is NOT used for topic readiness. Topic readiness is
	// handled explicitly above.
	// ------------------------------------------------------------

	select {
	case <-time.After(250 * time.Millisecond):
	case <-ctx.Done():
		t.Fatalf(
			"context cancelled while starting consumer: %v",
			ctx.Err(),
		)
	}

	// ------------------------------------------------------------
	// Source producer
	// ------------------------------------------------------------

	sourceProducer := NewProducer(
		brokers,
		sourceTopic,
	)

	t.Cleanup(func() {
		if err := sourceProducer.Close(); err != nil {
			t.Logf(
				"failed to close source producer: %v",
				err,
			)
		}
	})

	// ------------------------------------------------------------
	// Test message
	// ------------------------------------------------------------

	testKey := "DLQTEST-" + testID

	testValue := map[string]any{
		"type":    "invalid.dlq.test",
		"symbol":  "BTCUSDT",
		"message": "this message should go to the DLQ",
	}

	// ------------------------------------------------------------
	// Publish source message
	// ------------------------------------------------------------

	if err := sourceProducer.Publish(
		ctx,
		testKey,
		testValue,
	); err != nil {
		t.Fatalf(
			"failed to publish source message: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// DLQ reader
	// ------------------------------------------------------------

	reader := kafka.NewReader(
		kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   dlqTopic,

			GroupID: "velocity-dlq-reader-" +
				testID,

			StartOffset: kafka.FirstOffset,

			// Small fetch size is enough for this integration test.
			MinBytes: 1,
			MaxBytes: 10e6,
		},
	)

	t.Cleanup(func() {
		if err := reader.Close(); err != nil {
			t.Logf(
				"failed to close DLQ reader: %v",
				err,
			)
		}
	})

	// ------------------------------------------------------------
	// Read DLQ message
	// ------------------------------------------------------------

	var dlqMessage DLQMessage

	for {
		message, err := reader.ReadMessage(ctx)
		if err != nil {
			t.Fatalf(
				"failed to read DLQ message: %v",
				err,
			)
		}

		var candidate DLQMessage

		if err := json.Unmarshal(
			message.Value,
			&candidate,
		); err != nil {
			t.Fatalf(
				"failed to decode DLQ message: %v",
				err,
			)
		}

		// Ignore anything that does not belong to this test.
		if candidate.OriginalKey != testKey {
			continue
		}

		dlqMessage = candidate
		break
	}

	// ------------------------------------------------------------
	// Verify original topic
	// ------------------------------------------------------------

	if dlqMessage.OriginalTopic != sourceTopic {
		t.Fatalf(
			"expected original topic %s, got %s",
			sourceTopic,
			dlqMessage.OriginalTopic,
		)
	}

	// ------------------------------------------------------------
	// Verify original partition
	// ------------------------------------------------------------

	if dlqMessage.OriginalPartition < 0 {
		t.Fatalf(
			"expected valid original partition, got %d",
			dlqMessage.OriginalPartition,
		)
	}

	// ------------------------------------------------------------
	// Verify original offset
	// ------------------------------------------------------------

	if dlqMessage.OriginalOffset < 0 {
		t.Fatalf(
			"expected valid original offset, got %d",
			dlqMessage.OriginalOffset,
		)
	}

	// ------------------------------------------------------------
	// Verify original key
	// ------------------------------------------------------------

	if dlqMessage.OriginalKey != testKey {
		t.Fatalf(
			"expected original key %s, got %s",
			testKey,
			dlqMessage.OriginalKey,
		)
	}

	// ------------------------------------------------------------
	// Verify original value exists
	// ------------------------------------------------------------

	if len(dlqMessage.OriginalValue) == 0 {
		t.Fatal(
			"expected original message value",
		)
	}

	// ------------------------------------------------------------
	// Decode original value
	// ------------------------------------------------------------

	var originalValue map[string]any

	if err := json.Unmarshal(
		dlqMessage.OriginalValue,
		&originalValue,
	); err != nil {
		t.Fatalf(
			"failed to decode original message value: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// Verify original event type
	// ------------------------------------------------------------

	if originalValue["type"] != "invalid.dlq.test" {
		t.Fatalf(
			"expected original type invalid.dlq.test, got %v",
			originalValue["type"],
		)
	}

	// ------------------------------------------------------------
	// Verify original symbol
	// ------------------------------------------------------------

	if originalValue["symbol"] != "BTCUSDT" {
		t.Fatalf(
			"expected original symbol BTCUSDT, got %v",
			originalValue["symbol"],
		)
	}

	// ------------------------------------------------------------
	// Verify original message
	// ------------------------------------------------------------

	if originalValue["message"] !=
		"this message should go to the DLQ" {
		t.Fatalf(
			"unexpected original message: %v",
			originalValue["message"],
		)
	}

	// ------------------------------------------------------------
	// Verify failure reason
	// ------------------------------------------------------------

	if dlqMessage.Error != errTestMessage.Error() {
		t.Fatalf(
			"expected error %q, got %q",
			errTestMessage.Error(),
			dlqMessage.Error,
		)
	}

	// ------------------------------------------------------------
	// Verify failure timestamp
	// ------------------------------------------------------------

	if dlqMessage.FailedAt.IsZero() {
		t.Fatal(
			"expected FailedAt to be set",
		)
	}

	// Make sure the timestamp is sensible.
	if dlqMessage.FailedAt.After(time.Now().UTC()) {
		t.Fatalf(
			"expected FailedAt to be in the past, got %s",
			dlqMessage.FailedAt,
		)
	}

	// ------------------------------------------------------------
	// Success
	// ------------------------------------------------------------

	t.Logf(
		"DLQ verified: topic=%s partition=%d offset=%d key=%s error=%s",
		dlqMessage.OriginalTopic,
		dlqMessage.OriginalPartition,
		dlqMessage.OriginalOffset,
		dlqMessage.OriginalKey,
		dlqMessage.Error,
	)

	// ------------------------------------------------------------
	// Verify consumer remains alive
	// ------------------------------------------------------------

	select {
	case err := <-consumerDone:
		if err != nil {
			t.Fatalf(
				"consumer stopped unexpectedly: %v",
				err,
			)
		}

		t.Fatal(
			"consumer stopped unexpectedly after DLQ processing",
		)

	default:
		// Consumer is still running. This is expected.
	}
}
