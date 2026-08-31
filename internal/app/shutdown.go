package app

import (
	"time"

	"velocity/pkg/logger"
)

func Shutdown(container *Container) {
	if container.HTTP != nil {
		if err := container.HTTP.Shutdown(); err != nil {
			container.Logger.Error("http server shutdown error", logger.ErrorField(err))
		}
	}

	if container.KafkaEventPublisher != nil {
		// Give the publish goroutine a moment to drain whatever's
		// still queued before cutting it off - best effort, not
		// guaranteed, but better than dropping everything instantly.
		time.Sleep(500 * time.Millisecond)
		container.KafkaEventPublisher.Close()
	}

	if container.KafkaProducer != nil {
		if err := container.KafkaProducer.Close(); err != nil {
			container.Logger.Error(
				"kafka producer shutdown error",
				logger.ErrorField(err),
			)
		}
	}

	if container.DB != nil {
		container.DB.Close()
	}

	logger.Sync()
}
