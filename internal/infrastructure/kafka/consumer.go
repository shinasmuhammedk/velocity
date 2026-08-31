package kafka

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"

	"velocity/internal/infrastructure/metrics"
)

type MessageHandler func(
	ctx context.Context,
	message kafka.Message,
) error

type Consumer struct {
	reader  *kafka.Reader
	handler MessageHandler
	dlq     DLQPublisher
}

func NewConsumer(
	brokers []string,
	topic string,
	groupID string,
	handler MessageHandler,
	dlq DLQPublisher,
) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: groupID,

		// One message is fetched at a time for now.
		// We can tune this later for throughput.
		MinBytes: 1,
		MaxBytes: 10e6,
	})

	return &Consumer{
		reader:  reader,
		handler: handler,
		dlq:     dlq,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	for {
		message, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			metrics.KafkaConsumeFailures.Inc()

			return fmt.Errorf(
				"read kafka message: %w",
				err,
			)
		}

		metrics.KafkaMessagesConsumed.Inc()

		if c.handler == nil {
			continue
		}

		if err := c.handler(ctx, message); err != nil {
			if c.dlq == nil {
				metrics.KafkaConsumeFailures.Inc()

				return fmt.Errorf(
					"message handling failed and no DLQ configured: %w",
					err,
				)
			}

			if dlqErr := c.dlq.Publish(
				ctx,
				message,
				err,
			); dlqErr != nil {
				metrics.KafkaConsumeFailures.Inc()

				return fmt.Errorf(
					"publish failed message to DLQ: %w",
					dlqErr,
				)
			}

			fmt.Println(
				"KAFKA MESSAGE SENT TO DLQ:",
				"topic:", message.Topic,
				"partition:", message.Partition,
				"offset:", message.Offset,
				"key:", string(message.Key),
				"error:", err,
			)

			continue
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
