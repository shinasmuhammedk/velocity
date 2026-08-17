package kafka

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type MessageHandler func(ctx context.Context, message kafka.Message) error

type Consumer struct {
	reader  *kafka.Reader
	handler MessageHandler
}

func NewConsumer(
	brokers []string,
	topic string,
	groupID string,
	handler MessageHandler,
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
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	for {
		message, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			// A read failure means something is wrong with the
			// connection to Kafka itself (broker down, network issue,
			// etc). That's fatal — there's nothing more this consumer
			// can do, so it stops and lets the caller decide what to do
			// (e.g. restart the process).
			return fmt.Errorf("read kafka message: %w", err)
		}

		if c.handler == nil {
			continue
		}

		if err := c.handler(ctx, message); err != nil {
			// A handler failure means THIS ONE message was bad
			// (malformed JSON, unknown event type, etc) — not that
			// Kafka itself is broken. Log it and move on to the next
			// message instead of taking the whole consumer down.
			fmt.Println(
				"KAFKA MESSAGE HANDLING ERROR (skipped):",
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