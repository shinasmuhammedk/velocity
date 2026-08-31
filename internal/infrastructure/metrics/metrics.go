package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	OrdersSubmitted = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_orders_submitted_total",
			Help: "Total number of submitted orders",
		},
	)

	OrdersCancelled = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_orders_cancelled_total",
			Help: "Total number of cancelled orders",
		},
	)

	OrdersModified = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_orders_modified_total",
			Help: "Total number of modified orders",
		},
	)

	TradesExecuted = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_trades_executed_total",
			Help: "Total number of executed trades",
		},
	)

	// ------------------------------------------------------------
	// Kafka producer metrics
	// ------------------------------------------------------------

	KafkaMessagesProduced = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_kafka_messages_produced_total",
			Help: "Total number of Kafka messages successfully produced",
		},
	)

	KafkaProduceFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_kafka_produce_failures_total",
			Help: "Total number of Kafka message production failures",
		},
	)

	KafkaProducerRetries = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_kafka_producer_retries_total",
			Help: "Total number of Kafka producer retry attempts",
		},
	)

	// ------------------------------------------------------------
	// Kafka consumer metrics
	// ------------------------------------------------------------

	KafkaMessagesConsumed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_kafka_messages_consumed_total",
			Help: "Total number of Kafka messages successfully consumed",
		},
	)

	KafkaConsumeFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_kafka_consume_failures_total",
			Help: "Total number of Kafka consumer failures",
		},
	)

	KafkaDLQMessages = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_kafka_dlq_messages_total",
			Help: "Total number of Kafka messages successfully published to the DLQ",
		},
	)

	// ------------------------------------------------------------
	// Kafka health / readiness
	// ------------------------------------------------------------

	KafkaHealth = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "velocity_kafka_health_status",
			Help: "Kafka health status: 1 healthy, 0 unhealthy",
		},
	)

	KafkaReadiness = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "velocity_kafka_readiness_status",
			Help: "Kafka readiness status: 1 ready, 0 not ready",
		},
	)

	KafkaEventQueueDropped = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_kafka_event_queue_dropped_total",
			Help: "Total number of Kafka events dropped because the publisher queue was full",
		},
	)

	RateLimitAllowed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "velocity_rate_limit_allowed_total",
			Help: "Total number of requests allowed by the rate limiter",
		},
		[]string{"action"},
	)

	RateLimitRejected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "velocity_rate_limit_rejected_total",
			Help: "Total number of requests rejected by the rate limiter",
		},
		[]string{"action"},
	)

	RateLimitErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "velocity_rate_limit_errors_total",
			Help: "Total number of rate limiter infrastructure errors",
		},
		[]string{"action"},
	)

	registerOnce sync.Once
)

// Register registers all Velocity Prometheus metrics.
//
// sync.Once makes registration safe if Register is called
// multiple times during application startup or testing.
func Register() {
	registerOnce.Do(func() {
		prometheus.MustRegister(
			OrdersSubmitted,
			OrdersCancelled,
			OrdersModified,
			TradesExecuted,

			KafkaMessagesProduced,
			KafkaProduceFailures,
			KafkaProducerRetries,

			KafkaMessagesConsumed,
			KafkaConsumeFailures,
			KafkaDLQMessages,

			KafkaHealth,
			KafkaReadiness,

			KafkaEventQueueDropped,

			RateLimitAllowed,
			RateLimitRejected,
			RateLimitErrors,
		)
	})
}
