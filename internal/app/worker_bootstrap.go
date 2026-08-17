package app

import (
	"fmt"

	"go.uber.org/zap"

	"velocity/internal/config"
	"velocity/internal/engine/events"
	"velocity/internal/infrastructure/kafka"
	"velocity/pkg/logger"
)

type WorkerContainer struct {
	Config     *config.Config
	Logger     *zap.Logger
	Consumer   *kafka.Consumer
	Subscriber *kafka.EventSubscriber
	Dispatcher *events.Dispatcher
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

	consumer := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.Topic,
		cfg.Kafka.GroupID,
		subscriber.HandleMessage,
	)

	log.Info("kafka consumer initialized")

	return &WorkerContainer{
		Config:     cfg,
		Logger:     log,
		Consumer:   consumer,
		Subscriber: subscriber,
		Dispatcher: dispatcher,
	}, nil
}