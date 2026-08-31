package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	defaultTopicPartitions   = 1
	defaultReplicationFactor = 1

	topicReadyTimeout  = 10 * time.Second
	topicRetryInterval = 200 * time.Millisecond
)

type TopicConfig struct {
	Name              string
	NumPartitions     int
	ReplicationFactor int
}

func EnsureTopics(
	brokers []string,
	topics ...TopicConfig,
) error {
	if len(brokers) == 0 {
		return fmt.Errorf("kafka brokers are not configured")
	}

	conn, err := kafka.Dial(
		"tcp",
		brokers[0],
	)
	if err != nil {
		return fmt.Errorf("connect to kafka: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("get kafka controller: %w", err)
	}

	controllerAddr := fmt.Sprintf(
		"%s:%d",
		controller.Host,
		controller.Port,
	)

	controllerConn, err := kafka.Dial(
		"tcp",
		controllerAddr,
	)
	if err != nil {
		return fmt.Errorf(
			"connect to kafka controller: %w",
			err,
		)
	}
	defer controllerConn.Close()

	for _, topic := range topics {
		if topic.Name == "" {
			return fmt.Errorf(
				"kafka topic name cannot be empty",
			)
		}

		partitions := topic.NumPartitions
		if partitions <= 0 {
			partitions = defaultTopicPartitions
		}

		replicationFactor := topic.ReplicationFactor
		if replicationFactor <= 0 {
			replicationFactor = defaultReplicationFactor
		}

		if err := ensureTopic(
			controllerConn,
			topic.Name,
			partitions,
			replicationFactor,
		); err != nil {
			return err
		}

		if err := waitForTopicReady(
			brokers,
			topic.Name,
			partitions,
		); err != nil {
			return err
		}
	}

	return nil
}

func ensureTopic(
	conn *kafka.Conn,
	topic string,
	partitions int,
	replicationFactor int,
) error {
	_, err := conn.ReadPartitions(topic)

	if err == nil {
		return nil
	}

	if err := conn.CreateTopics(
		kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     partitions,
			ReplicationFactor: replicationFactor,
		},
	); err != nil {
		return fmt.Errorf(
			"create kafka topic %q: %w",
			topic,
			err,
		)
	}

	return nil
}

func waitForTopicReady(
	brokers []string,
	topic string,
	partitions int,
) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		topicReadyTimeout,
	)
	defer cancel()

	for {
		ready := true

		for partition := 0; partition < partitions; partition++ {
			conn, err := kafka.DialLeader(
				ctx,
				"tcp",
				brokers[0],
				topic,
				partition,
			)
			if err != nil {
				ready = false
				break
			}

			if err := conn.Close(); err != nil {
				return fmt.Errorf(
					"close kafka leader connection for topic %q: %w",
					topic,
					err,
				)
			}
		}

		if ready {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"kafka topic %q did not become ready within %s: %w",
				topic,
				topicReadyTimeout,
				ctx.Err(),
			)

		case <-time.After(topicRetryInterval):
		}
	}
}
