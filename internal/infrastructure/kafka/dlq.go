package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"

	"velocity/internal/infrastructure/metrics"
)

type DLQMessage struct {
	OriginalTopic     string    `json:"original_topic"`
	OriginalPartition int       `json:"original_partition"`
	OriginalOffset    int64     `json:"original_offset"`
	OriginalKey       string    `json:"original_key"`
	OriginalValue     []byte    `json:"original_value"`
	Error             string    `json:"error"`
	FailedAt          time.Time `json:"failed_at"`
}

type DLQPublisher interface {
	Publish(
		ctx context.Context,
		message kafka.Message,
		reason error,
	) error
}

type KafkaDLQPublisher struct {
	producer *Producer
}

func NewKafkaDLQPublisher(
	producer *Producer,
) *KafkaDLQPublisher {
	return &KafkaDLQPublisher{
		producer: producer,
	}
}

func (p *KafkaDLQPublisher) Publish(
	ctx context.Context,
	message kafka.Message,
	reason error,
) error {
	dlqMessage := DLQMessage{
		OriginalTopic:     message.Topic,
		OriginalPartition: message.Partition,
		OriginalOffset:    message.Offset,
		OriginalKey:       string(message.Key),
		OriginalValue:     message.Value,
		Error:             reason.Error(),
		FailedAt:          time.Now().UTC(),
	}

	err := p.producer.Publish(
		ctx,
		string(message.Key),
		dlqMessage,
	)
	if err != nil {
		return err
	}

	metrics.KafkaDLQMessages.Inc()

	return nil
}
