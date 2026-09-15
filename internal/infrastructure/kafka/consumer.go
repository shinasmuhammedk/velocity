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
		// FetchMessage - unlike ReadMessage - does NOT commit the
		// offset. The commit is deferred until after the message has
		// actually been handled (or its failure durably recorded in
		// the DLQ) below. This is the fix for the message-loss window
		// documented and reproduced in chaos_readmessage_commit_test.go:
		// under the old ReadMessage-based loop, the offset was already
		// committed before the handler ever ran, so a crash during
		// settlement silently and permanently lost the trade.
		message, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			metrics.KafkaConsumeFailures.Inc()

			return fmt.Errorf(
				"fetch kafka message: %w",
				err,
			)
		}

		metrics.KafkaMessagesConsumed.Inc()

		if c.handler == nil {
			if err := c.reader.CommitMessages(ctx, message); err != nil {
				metrics.KafkaConsumeFailures.Inc()

				return fmt.Errorf(
					"commit kafka message: %w",
					err,
				)
			}

			continue
		}

		if err := c.handler(ctx, message); err != nil {
			if c.dlq == nil {
				metrics.KafkaConsumeFailures.Inc()

				// Deliberately not committed: with no DLQ configured,
				// leaving the offset uncommitted means this message is
				// redelivered - to this consumer after a restart, or
				// to another member of the group - instead of being
				// silently dropped. Whoever operates this consumer is
				// expected to fix the handler or configure a DLQ
				// before it can make forward progress again.
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

				// Also deliberately not committed: if the DLQ publish
				// itself failed, the failure isn't durably recorded
				// anywhere yet. Leaving the offset uncommitted means
				// it's redelivered and retried, rather than lost -
				// exactly the failure mode this change exists to close.
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

			// Only commit now that the failure has been durably
			// recorded in the DLQ. A crash before this line means the
			// message is redelivered and sent to the DLQ again on the
			// next attempt - a harmless duplicate DLQ entry - rather
			// than being lost.
			if err := c.reader.CommitMessages(ctx, message); err != nil {
				metrics.KafkaConsumeFailures.Inc()

				return fmt.Errorf(
					"commit kafka message after dlq publish: %w",
					err,
				)
			}

			continue
		}

		// The handler succeeded - only now is it safe to commit. A
		// crash at any point before this line means the message was
		// never marked consumed, so it gets redelivered and the
		// handler runs again. For TradeConsumer specifically, that
		// redelivery is safe: settlementservice.Settle is idempotent
		// on trade ID (ON CONFLICT DO NOTHING).
		if err := c.reader.CommitMessages(ctx, message); err != nil {
			metrics.KafkaConsumeFailures.Inc()

			return fmt.Errorf(
				"commit kafka message: %w",
				err,
			)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}