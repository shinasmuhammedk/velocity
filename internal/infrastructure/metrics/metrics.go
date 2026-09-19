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

	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "velocity_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "route", "status"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "velocity_http_request_duration_seconds",
			Help: "HTTP request duration in seconds",
		},
		[]string{"method", "route", "status"},
	)

	// ------------------------------------------------------------
	// Settlement metrics
	// ------------------------------------------------------------

	SettlementsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_settlements_total",
			Help: "Total number of settlement attempts",
		},
	)

	SettlementFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_settlement_failures_total",
			Help: "Total number of settlement attempts that failed",
		},
	)

	SettlementDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "velocity_settlement_duration_seconds",
			Help: "Settlement execution duration in seconds",
		},
	)

	SettlementRetries = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_settlement_retries_total",
			Help: "Total number of failed-settlement retry attempts",
		},
	)

	FailedSettlementsCurrent = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "velocity_failed_settlements_current",
			Help: "Current number of unresolved and retryable failed settlements",
		},
	)

	FailedSettlementsDead = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_failed_settlements_dead_total",
			Help: "Total number of failed settlements successfully moved to dead state",
		},
	)

	FailedSettlementsRecovered = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_failed_settlements_recovered_total",
			Help: "Total number of failed settlements successfully recovered",
		},
	)

	// ------------------------------------------------------------
	// Redis market cache metrics
	// ------------------------------------------------------------

	MarketCacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "velocity_market_cache_hits_total",
			Help: "Total number of successful market cache reads",
		},
		[]string{"operation"},
	)

	MarketCacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "velocity_market_cache_misses_total",
			Help: "Total number of market cache misses",
		},
		[]string{"operation"},
	)

	MarketCacheErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "velocity_market_cache_errors_total",
			Help: "Total number of market cache infrastructure errors",
		},
		[]string{"operation"},
	)

	MarketCacheOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "velocity_market_cache_operation_duration_seconds",
			Help: "Market cache operation duration in seconds",
		},
		[]string{"operation"},
	)

	UserStreamDeliveryFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "velocity_userstream_delivery_failures_total",
			Help: "Total number of user-stream message delivery failures",
		},
		[]string{"event_type"},
	)

	// ------------------------------------------------------------
	// Matching engine
	//
	// EngineCommandDuration measures end-to-end command latency as the
	// caller experiences it: enqueue -> single-threaded processing ->
	// result. That includes queue wait time, which is deliberate - a
	// backed-up command queue is exactly the condition these histograms
	// exist to surface.
	// ------------------------------------------------------------

	EngineCommandsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "velocity_engine_commands_total",
			Help: "Total number of engine commands processed",
		},
		[]string{"symbol", "kind", "outcome"},
	)

	EngineCommandDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "velocity_engine_command_duration_seconds",
			Help: "Engine command latency (enqueue to result) in seconds",
			// Microsecond-scale buckets: the hot path targets are in
			// docs/performance/performance-targets.md, and the default
			// Prometheus buckets start at 5ms, which is far too coarse
			// to see anything here.
			Buckets: []float64{
				0.000005, 0.00001, 0.000025, 0.00005,
				0.0001, 0.00025, 0.0005, 0.001,
				0.005, 0.01, 0.05, 0.1, 0.5, 1,
			},
		},
		[]string{"symbol", "kind"},
	)

	EngineQueueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "velocity_engine_queue_depth",
			Help: "Current depth of the engine command queue",
		},
		[]string{"symbol"},
	)

	EngineQueueCapacity = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "velocity_engine_queue_capacity",
			Help: "Capacity of the engine command queue",
		},
		[]string{"symbol"},
	)

	EngineTradeQueueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "velocity_engine_trade_queue_depth",
			Help: "Current depth of the engine outbound trade queue",
		},
		[]string{"symbol"},
	)

	EngineSequence = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "velocity_engine_sequence",
			Help: "Current engine sequence number",
		},
		[]string{"symbol"},
	)

	OrderBookResting = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "velocity_orderbook_resting_orders",
			Help: "Number of resting orders currently in the order book",
		},
		[]string{"symbol", "side"},
	)

	OrderBookBestPrice = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "velocity_orderbook_best_price",
			Help: "Best bid/ask price currently in the order book, in ticks",
		},
		[]string{"symbol", "side"},
	)

	StopBookResting = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "velocity_stopbook_resting_orders",
			Help: "Number of untriggered stop orders held per symbol",
		},
		[]string{"symbol"},
	)

	// ------------------------------------------------------------
	// Engine registry
	// ------------------------------------------------------------

	EnginesActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "velocity_engines_active",
			Help: "Number of matching engines currently resident in the registry",
		},
	)

	// ------------------------------------------------------------
	// Write-ahead log
	// ------------------------------------------------------------

	WALWritesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_wal_writes_total",
			Help: "Total number of WAL records durably written",
		},
	)

	WALWriteFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "velocity_wal_write_failures_total",
			Help: "Total number of WAL write failures",
		},
		[]string{"stage"},
	)

	WALWriteDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "velocity_wal_write_duration_seconds",
			Help: "WAL append + fsync duration in seconds",
			Buckets: []float64{
				0.0001, 0.00025, 0.0005, 0.001, 0.0025,
				0.005, 0.01, 0.025, 0.05, 0.1, 0.5, 1,
			},
		},
	)

	WALBytesWritten = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_wal_bytes_written_total",
			Help: "Total number of bytes appended to the WAL",
		},
	)

	// ------------------------------------------------------------
	// Snapshots
	// ------------------------------------------------------------

	SnapshotsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_snapshots_written_total",
			Help: "Total number of engine snapshots successfully written",
		},
	)

	SnapshotFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_snapshot_failures_total",
			Help: "Total number of engine snapshot write failures",
		},
	)

	SnapshotDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "velocity_snapshot_duration_seconds",
			Help:    "Engine snapshot capture + write duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	SnapshotLastSuccessTimestamp = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "velocity_snapshot_last_success_timestamp_seconds",
			Help: "Unix timestamp of the last successful snapshot write",
		},
	)

	// ------------------------------------------------------------
	// Recovery
	// ------------------------------------------------------------

	RecoveryDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "velocity_recovery_duration_seconds",
			Help:    "Startup recovery duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	RecoveredOrders = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_recovered_orders_total",
			Help: "Total number of open orders restored during recovery",
		},
	)

	RecoveryFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "velocity_recovery_failures_total",
			Help: "Total number of recovery failures",
		},
	)

	// ------------------------------------------------------------
	// Build info
	// ------------------------------------------------------------

	BuildInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "velocity_build_info",
			Help: "Build and runtime metadata, always 1. Use the labels.",
		},
		[]string{"version", "environment", "component"},
	)

	registerOnce sync.Once
)

// SetBuildInfo publishes static process metadata as a constant gauge so
// that dashboards and alerts can distinguish api from worker, and one
// deployed version from another, without a separate service discovery
// label.
func SetBuildInfo(version, environment, component string) {
	BuildInfo.
		WithLabelValues(version, environment, component).
		Set(1)
}

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

			HTTPRequestsTotal,
			HTTPRequestDuration,

			SettlementsTotal,
			SettlementFailures,
			SettlementDuration,
			SettlementRetries,
			FailedSettlementsCurrent,
			FailedSettlementsDead,
			FailedSettlementsRecovered,

			MarketCacheHits,
			MarketCacheMisses,
			MarketCacheErrors,
			MarketCacheOperationDuration,

			UserStreamDeliveryFailures,

			EngineCommandsTotal,
			EngineCommandDuration,
			EngineQueueDepth,
			EngineQueueCapacity,
			EngineTradeQueueDepth,
			EngineSequence,
			OrderBookResting,
			OrderBookBestPrice,
			StopBookResting,

			EnginesActive,

			WALWritesTotal,
			WALWriteFailures,
			WALWriteDuration,
			WALBytesWritten,

			SnapshotsTotal,
			SnapshotFailures,
			SnapshotDuration,
			SnapshotLastSuccessTimestamp,

			RecoveryDuration,
			RecoveredOrders,
			RecoveryFailures,

			BuildInfo,
		)
	})
}