package app

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"velocity/internal/config"
	"velocity/internal/engine/events"
	"velocity/internal/infrastructure/kafka"
	"velocity/internal/infrastructure/metrics"
	"velocity/pkg/logger"
)

type WorkerContainer struct {
	Config      *config.Config
	Logger      *zap.Logger
	Consumer    *kafka.Consumer
	Subscriber  *kafka.EventSubscriber
	Dispatcher  *events.Dispatcher
	DLQProducer *kafka.Producer

	// MetricsServer exposes this process's own scrape endpoint.
	//
	// The worker runs the Kafka consumer, the DLQ publisher and the
	// consumer health checker, all of which increment collectors in
	// internal/infrastructure/metrics. Without registering those
	// collectors and serving them from this process, every one of those
	// increments lands in an unregistered collector inside the worker
	// and is never scraped by anyone - the API process serves a
	// different copy of the same process-local registry.
	MetricsServer *metrics.Server
}

func WorkerBootstrap() (*WorkerContainer, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if err := logger.Init(cfg.App.Environment); err != nil {
		return nil, fmt.Errorf(
			"initialize logger: %w",
			err,
		)
	}

	log := logger.Logger()
	log.Info("worker configuration loaded")

	// Metrics: register collectors and expose them on the worker's own
	// port, which must differ from the API's (see MetricsConfig).
	metrics.Register()
	metrics.SetBuildInfo(
		cfg.App.Version,
		cfg.App.Environment,
		"worker",
	)

	metricsServer := metrics.NewServer(metrics.Options{
		Enabled: cfg.Metrics.Enabled,
		Host:    cfg.Metrics.Host,
		Port:    cfg.Metrics.WorkerPort,
		Path:    cfg.Metrics.Path,
	})

	if err := metricsServer.Start(); err != nil {
		return nil, fmt.Errorf("start worker metrics server: %w", err)
	}

	if metricsServer.Enabled() {
		log.Info(
			"prometheus metrics registered and exposed",
			zap.String("endpoint", metricsServer.Address()),
		)
	}

	err = kafka.EnsureTopics(
		cfg.Kafka.Brokers,
		kafka.TopicConfig{
			Name:              cfg.Kafka.Topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
		kafka.TopicConfig{
			Name:              cfg.Kafka.DLQTopic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"ensure kafka topics: %w",
			err,
		)
	}

	log.Info("kafka topics verified")

	dispatcher := events.NewDispatcher()

	audit := NewEventAuditSubscriber()
	dispatcher.Subscribe(events.TradeExecutedEventType, audit)
	dispatcher.Subscribe(events.OrderAcceptedEventType, audit)
	dispatcher.Subscribe(events.OrderRejectedEventType, audit)
	dispatcher.Subscribe(events.OrderCancelledEventType, audit)
	dispatcher.Subscribe(events.OrderModifiedEventType, audit)
	dispatcher.Subscribe(events.OrderTriggeredEventType, audit)

	subscriber := kafka.NewEventSubscriber(
		dispatcher,
	)

	dlqProducer := kafka.NewProducer(
		cfg.Kafka.Brokers,
		cfg.Kafka.DLQTopic,
	)

	dlq := kafka.NewKafkaDLQPublisher(
		dlqProducer,
	)

	consumer := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.Topic,
		cfg.Kafka.GroupID,
		subscriber.HandleMessage,
		dlq,
	)

	log.Info("kafka consumer initialized")

	return &WorkerContainer{
		Config:        cfg,
		Logger:        log,
		Consumer:      consumer,
		Subscriber:    subscriber,
		Dispatcher:    dispatcher,
		DLQProducer:   dlqProducer,
		MetricsServer: metricsServer,
	}, nil
}

// Shutdown releases the worker's resources in reverse dependency order.
func (c *WorkerContainer) Shutdown() {
	if c.DLQProducer != nil {
		if err := c.DLQProducer.Close(); err != nil {
			c.Logger.Error(
				"dlq producer shutdown error",
				zap.Error(err),
			)
		}
	}

	if c.MetricsServer != nil {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := c.MetricsServer.Shutdown(ctx); err != nil {
			c.Logger.Error(
				"metrics server shutdown error",
				zap.Error(err),
			)
		}
	}

	logger.Sync()
}