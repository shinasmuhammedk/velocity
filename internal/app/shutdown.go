package app

import (
	"context"
	"time"

	"velocity/pkg/logger"
)

func Shutdown(container *Container) {
	// 1. Stop accepting new HTTP requests
	if container.HTTP != nil {
		if err := container.HTTP.Shutdown(); err != nil {
			container.Logger.Error(
				"http server shutdown error",
				logger.ErrorField(err),
			)
		}
	}

	// 2. Stop gRPC server gracefully
	if container.GRPCServer != nil {
		container.GRPCServer.Stop()
	}

	// 3. Cancel application-wide background workers
	if container.ShutdownCancel != nil {
		container.ShutdownCancel()
	}

	// 4. Stop matching engines and trade consumers
	if container.Registry != nil {
		if err := container.Registry.Shutdown(); err != nil {
			container.Logger.Error(
				"engine registry shutdown error",
				logger.ErrorField(err),
			)
		}
	}

	// 5. Give queued Kafka events a moment to drain
	if container.KafkaEventPublisher != nil {
		time.Sleep(500 * time.Millisecond)
		container.KafkaEventPublisher.Close()
	}

	// 6. Close Kafka producer
	if container.KafkaProducer != nil {
		if err := container.KafkaProducer.Close(); err != nil {
			container.Logger.Error(
				"kafka producer shutdown error",
				logger.ErrorField(err),
			)
		}
	}

	// 7. Close Identity gRPC client
	if container.IdentityClient != nil {
		if err := container.IdentityClient.Close(); err != nil {
			container.Logger.Error(
				"identity client shutdown error",
				logger.ErrorField(err),
			)
		}
	}

	// 8. Close Redis
	if container.Redis != nil {
		if err := container.Redis.Close(); err != nil {
			container.Logger.Error(
				"redis shutdown error",
				logger.ErrorField(err),
			)
		}
	}

	// 9. Close PostgreSQL
	if container.DB != nil {
		container.DB.Close()
	}

	// 10. Stop the metrics listener.
	//
	// Deliberately last among the servers: keeping the scrape endpoint
	// alive through the rest of the drain means the final counter
	// values - queued events dropped, settlements failed on the way
	// down - are still scrapable while shutdown is in progress.
	if container.MetricsServer != nil {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := container.MetricsServer.Shutdown(ctx); err != nil {
			container.Logger.Error(
				"metrics server shutdown error",
				logger.ErrorField(err),
			)
		}
	}

	// 11. Flush logger
	logger.Sync()
}