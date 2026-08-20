package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

var errTestMessage = errors.New(
	"intentional DLQ test failure",
)

func TestKafkaDLQ_EndToEnd(t *testing.T) {
	brokers := []string{"localhost:9092"}

	// Dedicated topics for this integration test.
	// Do not use the production topics here because they may
	// already contain old messages.
	sourceTopic := "velocity-events-dlq-test"
	dlqTopic := "velocity-events-dlq-test-dlq"

	ctx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	// ------------------------------------------------------------
	// DLQ producer
	// ------------------------------------------------------------

	dlqProducer := NewProducer(
		brokers,
		dlqTopic,
	)
	defer dlqProducer.Close()

	dlqPublisher := NewKafkaDLQPublisher(
		dlqProducer,
	)

	// ------------------------------------------------------------
	// Consumer
	// ------------------------------------------------------------

	consumer := NewConsumer(
		brokers,
		sourceTopic,
		"velocity-dlq-test-"+time.Now().Format(
			"20060102150405.000000000",
		),
		func(
			ctx context.Context,
			message kafka.Message,
		) error {
			// Deliberately fail the message so that the consumer
			// sends it to the DLQ.
			return errTestMessage
		},
		dlqPublisher,
	)

	defer consumer.Close()

	consumerDone := make(chan error, 1)

	go func() {
		consumerDone <- consumer.Start(ctx)
	}()

	// ------------------------------------------------------------
	// Give consumer time to connect
	// ------------------------------------------------------------

	time.Sleep(1 * time.Second)

	// ------------------------------------------------------------
	// Source producer
	// ------------------------------------------------------------

	sourceProducer := NewProducer(
		brokers,
		sourceTopic,
	)
	defer sourceProducer.Close()

	testKey := "DLQTEST-" + time.Now().Format(
		"20060102150405.000000000",
	)

	testValue := map[string]any{
		"type":    "invalid.dlq.test",
		"symbol":  "BTCUSDT",
		"message": "this message should go to the DLQ",
	}

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
	// Read DLQ
	// ------------------------------------------------------------

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   dlqTopic,

		GroupID: "velocity-dlq-reader-" +
			time.Now().Format(
				"20060102150405.000000000",
			),

		StartOffset: kafka.FirstOffset,
	})

	defer reader.Close()

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

		// Ignore unrelated messages if the test DLQ already
		// contains anything from a previous test run.
		if candidate.OriginalKey != testKey {
			continue
		}

		dlqMessage = candidate
		break
	}

	// ------------------------------------------------------------
	// Verify DLQ metadata
	// ------------------------------------------------------------

	if dlqMessage.OriginalTopic != sourceTopic {
		t.Fatalf(
			"expected original topic %s, got %s",
			sourceTopic,
			dlqMessage.OriginalTopic,
		)
	}

	if dlqMessage.OriginalPartition < 0 {
		t.Fatalf(
			"expected valid original partition, got %d",
			dlqMessage.OriginalPartition,
		)
	}

	if dlqMessage.OriginalOffset < 0 {
		t.Fatalf(
			"expected valid original offset, got %d",
			dlqMessage.OriginalOffset,
		)
	}

	if dlqMessage.OriginalKey != testKey {
		t.Fatalf(
			"expected original key %s, got %s",
			testKey,
			dlqMessage.OriginalKey,
		)
	}

	if len(dlqMessage.OriginalValue) == 0 {
		t.Fatal("expected original message value")
	}

	// ------------------------------------------------------------
	// Verify original payload
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

	if originalValue["type"] != "invalid.dlq.test" {
		t.Fatalf(
			"expected original type invalid.dlq.test, got %v",
			originalValue["type"],
		)
	}

	if originalValue["symbol"] != "BTCUSDT" {
		t.Fatalf(
			"expected original symbol BTCUSDT, got %v",
			originalValue["symbol"],
		)
	}

	if originalValue["message"] != "this message should go to the DLQ" {
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
	// Verify timestamp
	// ------------------------------------------------------------

	if dlqMessage.FailedAt.IsZero() {
		t.Fatal("expected FailedAt to be set")
	}

	// ------------------------------------------------------------
	// Final verification
	// ------------------------------------------------------------

	t.Logf(
		"DLQ verified: topic=%s partition=%d offset=%d key=%s error=%s",
		dlqMessage.OriginalTopic,
		dlqMessage.OriginalPartition,
		dlqMessage.OriginalOffset,
		dlqMessage.OriginalKey,
		dlqMessage.Error,
	)

	// The consumer should still be running after successfully
	// moving the failed message to the DLQ.
	select {
	case err := <-consumerDone:
		if err != nil {
			t.Fatalf(
				"consumer stopped unexpectedly: %v",
				err,
			)
		}
	default:
	}
}
