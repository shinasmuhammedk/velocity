package app

import "velocity/pkg/logger"

func Shutdown(container *Container) {
	if container.HTTP != nil {
		if err := container.HTTP.Shutdown(); err != nil {
			container.Logger.Error("http server shutdown error", logger.ErrorField(err))
		}
	}

	if container.DB != nil {
		container.DB.Close()
	}

	logger.Sync()
}