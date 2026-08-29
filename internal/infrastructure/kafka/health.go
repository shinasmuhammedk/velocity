package kafka

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"

	"velocity/internal/infrastructure/metrics"
)

type HealthChecker struct {
	brokers []string
	topics  []string
}

func NewHealthChecker(
	brokers []string,
	topics ...string,
) *HealthChecker {
	return &HealthChecker{
		brokers: brokers,
		topics:  topics,
	}
}

// Check verifies that Kafka itself is reachable.
func (h *HealthChecker) Check(
	ctx context.Context,
) error {
	if len(h.brokers) == 0 {
		metrics.KafkaHealth.Set(0)

		return fmt.Errorf(
			"kafka brokers are not configured",
		)
	}

	conn, err := kafka.DialContext(
		ctx,
		"tcp",
		h.brokers[0],
	)
	if err != nil {
		metrics.KafkaHealth.Set(0)

		return fmt.Errorf(
			"connect to kafka: %w",
			err,
		)
	}
	defer conn.Close()

	metrics.KafkaHealth.Set(1)

	return nil
}

// Readiness verifies that Kafka is reachable and all required
// Velocity topics are available.
func (h *HealthChecker) Readiness(
	ctx context.Context,
) error {
	if len(h.brokers) == 0 {
		metrics.KafkaHealth.Set(0)
		metrics.KafkaReadiness.Set(0)

		return fmt.Errorf(
			"kafka brokers are not configured",
		)
	}

	conn, err := kafka.DialContext(
		ctx,
		"tcp",
		h.brokers[0],
	)
	if err != nil {
		metrics.KafkaHealth.Set(0)
		metrics.KafkaReadiness.Set(0)

		return fmt.Errorf(
			"connect to kafka for readiness check: %w",
			err,
		)
	}

	defer conn.Close()

	// Kafka itself is reachable.
	metrics.KafkaHealth.Set(1)

	for _, topic := range h.topics {
		if topic == "" {
			continue
		}

		partitions, err := conn.ReadPartitions(topic)
		if err != nil {
			metrics.KafkaReadiness.Set(0)

			return fmt.Errorf(
				"read kafka topic %q metadata: %w",
				topic,
				err,
			)
		}

		if len(partitions) == 0 {
			metrics.KafkaReadiness.Set(0)

			return fmt.Errorf(
				"kafka topic %q has no partitions",
				topic,
			)
		}
	}

	metrics.KafkaReadiness.Set(1)

	return nil
}