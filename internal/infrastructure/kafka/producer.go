package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"velocity/internal/infrastructure/metrics"
)

const (
	producerMaxAttempts = 10
	producerRetryDelay  = 500 * time.Millisecond
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(
	brokers []string,
	topic string,
) *Producer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.Hash{},
	}

	return &Producer{
		writer: writer,
	}
}

func (p *Producer) Publish(
	ctx context.Context,
	key string,
	value any,
) error {
	data, err := json.Marshal(value)
	if err != nil {
		metrics.KafkaProduceFailures.Inc()

		return fmt.Errorf(
			"marshal kafka message: %w",
			err,
		)
	}

	var lastErr error

	for attempt := 1; attempt <= producerMaxAttempts; attempt++ {
		err = p.writer.WriteMessages(
			ctx,
			kafka.Message{
				Key:   []byte(key),
				Value: data,
			},
		)

		if err == nil {
			metrics.KafkaMessagesProduced.Inc()

			return nil
		}

		lastErr = err

		// Count every retry after the initial failed attempt.
		if attempt < producerMaxAttempts {
			metrics.KafkaProducerRetries.Inc()
		}

		select {
		case <-ctx.Done():
			metrics.KafkaProduceFailures.Inc()

			return fmt.Errorf(
				"publish kafka message: %w",
				ctx.Err(),
			)

		case <-time.After(producerRetryDelay):
		}
	}

	metrics.KafkaProduceFailures.Inc()

	return fmt.Errorf(
		"publish kafka message after %d attempts: %w",
		producerMaxAttempts,
		lastErr,
	)
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
