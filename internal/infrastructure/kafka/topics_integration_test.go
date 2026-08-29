package kafka

import (
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestEnsureTopics(t *testing.T) {
	brokers := []string{"localhost:9092"}

	topicName := "velocity-topic-provision-test-" +
		time.Now().Format("20060102150405.000000000")

	err := EnsureTopics(
		brokers,
		TopicConfig{
			Name:              topicName,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to ensure Kafka topic: %v",
			err,
		)
	}

	conn, err := kafka.Dial(
		"tcp",
		brokers[0],
	)
	if err != nil {
		t.Fatalf(
			"failed to connect to Kafka: %v",
			err,
		)
	}
	defer conn.Close()

	defer func() {
		if err := conn.DeleteTopics(topicName); err != nil {
			t.Logf(
				"failed to delete test topic %s: %v",
				topicName,
				err,
			)
		}
	}()

	partitions, err := conn.ReadPartitions(topicName)
	if err != nil {
		t.Fatalf(
			"failed to read created topic: %v",
			err,
		)
	}

	if len(partitions) != 1 {
		t.Fatalf(
			"expected 1 partition, got %d",
			len(partitions),
		)
	}

	t.Logf(
		"Kafka topic provisioned successfully: %s",
		topicName,
	)
}
