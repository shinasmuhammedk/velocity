package app

import "velocity/pkg/logger"

func Shutdown(container *Container) {

	if container.DB != nil {
		container.DB.Close()
	}

	logger.Sync()
}